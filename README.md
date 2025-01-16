# Detect

go + opencv pedestrian detection

## Requirements
1. go 1.23.4
1. openvino 2024.6+
1. opencv 4.11+
1. Intel CPU or Intel GPU or an external Intel VPU supported by openVINO

## Dependencies
If your package manager provided opencv installation isn't built with openVINO support or doesn't support openvino 2024.6+, you have to build opencv from source against the specific version of openvino.
1. Install openVINO 2024.6: https://docs.openvino.ai/2024/get-started/install-openvino.html (Building from source seems to be broken on non-ubuntu systems, but you can use a binary package with C++ bindings support)
1. Clone the opencv repository `git clone --recurse-submodules https://github.com/opencv/opencv.git`
1. Use the provided build script, install the dependencies if missing `./opencv-against-openvino-build-script.sh` (WIP, edit the paths in the script to match your system)

## Build instructions
1. `git clone git@github.com:Robogera/detect.git`
1. `cd detect/`
1. `go mod tidy`
1. `mkdir bin`
1. `cd src/`
1. Set the include paths for CGO compiler:
1.1. bash: `source prepare-opencv-openvino-build-env.sh`
1.1. fish: `bass source prepare-opencv-openvino-build-env` 
1. `go build -o bin/detect`
