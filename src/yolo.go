package main

import (
	"image"
	"image/color"

	"github.com/Robogera/detect/pkg/config"
	"gocv.io/x/gocv"
)

func getOutputLayerNames(net *gocv.Net) []string {
	var output_layer_names []string
	for _, i := range net.GetUnconnectedOutLayers() {
		layer := net.GetLayer(i)
		name := layer.GetName()
		if name != "_input" {
			output_layer_names = append(output_layer_names, name)
		}
	}
	return output_layer_names
}

func detectObjects(net *gocv.Net, img *gocv.Mat, cfg *config.ConfigFile, output_layer_names []string) (*gocv.Mat, error) {
	// profile this and maybe don't clone
	cloned_img := gocv.NewMat()
	img.ConvertTo(&cloned_img, gocv.MatTypeCV32F) // No idea which format to use
	blob_conv_params := gocv.NewImageToBlobParams(
		1.0/cfg.Model.ScaleFactor,
		image.Pt(int(cfg.Model.X), int(cfg.Model.Y)),
		gocv.NewScalar(0, 0, 0, 0),
		true,
		gocv.MatTypeCV32F,
		gocv.DataLayoutNCHW,
		gocv.PaddingModeLetterbox,
		gocv.NewScalar(0, 0, 0, 0),
	)
	blob := gocv.BlobFromImageWithParams(cloned_img, blob_conv_params)
	defer blob.Close()

	net.SetInput(blob, "")

	outputs := net.ForwardLayers(output_layer_names)
	defer func() {
		for _, output := range outputs {
			output.Close()
		}
	}()

	// YOLO-models authored by ultralythics are transposed for some reason
	// this is required to unfuck them
	// (seems to be in place and zero performance cost so ok)
	if cfg.Model.Transpose {
		gocv.TransposeND(outputs[0], []int{0, 2, 1}, &outputs[0])
	}

	for _, output := range outputs {
		output_2d := output.Reshape(1, output.Size()[1])
		cols := output_2d.Cols()
		var boxes []image.Rectangle
		var confidences []float32
		for i := 0; i < output_2d.Rows(); i++ {
			func() {
				row := output_2d.RowRange(i, i+1)
				defer row.Close()
				// values at indexes 4:cols are the confidence scores of the
				// object classes
				confidence_scores_area := row.ColRange(4, cols)
				defer confidence_scores_area.Close()
				_, confidence, _, class_id := gocv.MinMaxLoc(confidence_scores_area)
				// drop everything that isn't most likely a person
				if class_id.X != int(cfg.Model.PersonClassIndex) {
					return
				}
				// elements 0 and 1 correspond to the bounding box center coordinates
				x, y := int(row.GetFloatAt(0, 0)), int(row.GetFloatAt(0, 1))
				// and elements 2 and 3 are the box dimensions
				half_w, half_h := int(row.GetFloatAt(0, 2)/2.0), int(row.GetFloatAt(0, 3)/2.0)
				boxes = append(boxes, image.Rect(x-half_w, y-half_h, x+half_w, y+half_h))
				confidences = append(confidences, confidence)
			}()
		}
		output_2d.Close()

    boxes = blob_conv_params.BlobRectsToImageRects(boxes, image.Pt(cloned_img.Cols(), cloned_img.Rows()))

		for _, i := range gocv.NMSBoxes(boxes, confidences, cfg.Model.ConfidenceThreshold, cfg.Model.NMSThreshold) {
			gocv.Rectangle(&cloned_img, boxes[i], color.RGBA{255, 0, 0, 255}, 1)
		}

	}

	return &cloned_img, nil
}
