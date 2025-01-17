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
	cloned_img := img.Clone()
	img.ConvertTo(&cloned_img, gocv.MatTypeCV32F) // No idea which format to use
	blob := gocv.BlobFromImage(
		cloned_img,
		1.0/cfg.Model.ScaleFactor,
		image.Pt(int(cfg.Model.X), int(cfg.Model.Y)),
		gocv.NewScalar(0, 0, 0, 0),
		true,
		false)
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
		output = output.Reshape(1, output.Size()[1])
		cols := output.Cols()
		var boxes []image.Rectangle
		var confidences []float32
		for i := 0; i < output.Rows(); i++ {
			row := output.RowRange(i, i+1)
      // values at indexes 4:cols are the confidence scores of the
      // object classes
			_, confidence, _, class_id := gocv.MinMaxLoc(row.ColRange(4, cols))
      // drop everything that isn't most likely a person
			if class_id.X != int(cfg.Model.PersonClassIndex) {
				continue
			}
      // elements 0 and 1 correspond to the bounding box center coordinates
			x, y := int(row.GetFloatAt(0, 0)), int(row.GetFloatAt(0, 1))
      // and elements 2 and 3 are the box dimensions 
			half_w, half_h := int(row.GetFloatAt(0, 2)/2.0), int(row.GetFloatAt(0, 3)/2.0)

			boxes = append(boxes, image.Rect(x-half_w, y-half_h, x+half_w, y+half_h))
			confidences = append(confidences, confidence)
		}

		for _, i := range gocv.NMSBoxes(boxes, confidences, cfg.Model.ConfidenceThreshold, cfg.Model.NMSThreshold) {
			gocv.Rectangle(&cloned_img, boxes[i], color.RGBA{255, 0, 0, 255}, 1)
		}

	}

	return &cloned_img, nil
}