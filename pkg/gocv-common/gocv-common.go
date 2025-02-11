package gocvcommon

import "gocv.io/x/gocv"

func GetOutputLayerNames(net *gocv.Net) []string {
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

func CheckLayerName(net *gocv.Net, layer_name string) bool {
	for _, i := range net.GetUnconnectedOutLayers() {
		layer := net.GetLayer(i)
		name := layer.GetName()
		if name == layer_name {
			return true
		}
	}
	return false
}
