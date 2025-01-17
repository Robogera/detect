package main

import (
	"image"
	"image/color"
	"log"

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

func detectObjects(net *gocv.Net, img *gocv.Mat, cfg *config.ConfigFile, output_layer_names []string) (*gocv.Mat, []string, error) {
	cloned_img := img.Clone()
	img.ConvertTo(&cloned_img, gocv.MatTypeCV32F) // No idea which format to use
	blob := gocv.BlobFromImage(
		cloned_img,
		1/255.0,
		image.Pt(640, 640),
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
			_, confidence, _, class_id := gocv.MinMaxLoc(row.ColRange(4, cols))
			if confidence > 1.0 {
				log.Println("nigga please")
			}
			if class_id.X != 1 {
				continue
			}
			x, y := int(row.GetFloatAt(0, 0)), int(row.GetFloatAt(0, 1))
			half_w, half_h := int(row.GetFloatAt(0, 2)/2.0), int(row.GetFloatAt(0, 3)/2.0)

			boxes = append(boxes, image.Rect(x-half_w, y-half_h, x+half_w, y+half_h))
			confidences = append(confidences, confidence)
		}

		for _, i := range gocv.NMSBoxes(boxes, confidences, 0.995, 0.05) {
			gocv.Rectangle(&cloned_img, boxes[i], color.RGBA{255, 0, 0, 255}, 3)
		}

	}

	return &cloned_img, nil, nil
}
