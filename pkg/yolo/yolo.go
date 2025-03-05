package yolo

import (
	"image"

	"github.com/Robogera/detect/pkg/config"
	"gocv.io/x/gocv"
)

func Detect(net *gocv.Net, img *gocv.Mat, cfg *config.ConfigFile, output_layer_names []string, params *gocv.ImageToBlobParams) ([]image.Rectangle, error) {
	blob := gocv.BlobFromImageWithParams(*img, *params)
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
	if cfg.Yolo.Transpose {
		gocv.TransposeND(outputs[0], []int{0, 2, 1}, &outputs[0])
	}

	var nms_boxes []image.Rectangle

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
				if class_id.X != int(cfg.Yolo.PersonClassIndex) {
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

		if len(boxes) > 0 {
			indices := gocv.NMSBoxes(boxes, confidences, cfg.Yolo.ConfidenceThreshold, cfg.Yolo.NMSThreshold)

			nms_boxes = make([]image.Rectangle, len(indices))
			for i, j := range indices {
				nms_boxes[i] = boxes[j]
			}
			if len(nms_boxes) > 0 {

				nms_boxes = params.BlobRectsToImageRects(nms_boxes, image.Pt(img.Cols(), img.Rows()))
			}
		}
	}

	return nms_boxes, nil
}
