package enums

// declaration of various enums for
// user data validation purposes
// PS. I HATE MY LIFE, should have used haskell

import (
	"github.com/orsinium-labs/enum"
)

type ModelFormat enum.Member[string]

var (
	mf = enum.NewBuilder[string, ModelFormat]()

	ModelONNX     = mf.Add(ModelFormat{"onnx"})
	ModelOpenVINO = mf.Add(ModelFormat{"openvino"})
	ModelCaffe    = mf.Add(ModelFormat{"caffe"})

	ModelFormats = mf.Enum()
)

type DeviceType enum.Member[string]

var (
	dt = enum.NewBuilder[string, DeviceType]()

	DeviceCPU = dt.Add(DeviceType{"cpu"})
	DeviceGPU = dt.Add(DeviceType{"gpu"})
	DeviceVPU = dt.Add(DeviceType{"vpu"})

	DeviceTypes = dt.Enum()
)

type InputType enum.Member[string]

var (
	ifl = enum.NewBuilder[string, InputType]()

	InputFile   = ifl.Add(InputType{"file"})
	InputWebcam = ifl.Add(InputType{"webcam"})

	InputTypes = ifl.Enum()
)

type LoggingLevel enum.Member[string]

var (
	ll = enum.NewBuilder[string, LoggingLevel]()

	LoggingLevelDebug = ll.Add(LoggingLevel{"debug"})
	LoggingLevelInfo  = ll.Add(LoggingLevel{"info"})
	LoggingLevelWarn  = ll.Add(LoggingLevel{"warn"})
	LoggingLevelError = ll.Add(LoggingLevel{"error"})

	LoggingLevels = ll.Enum()
)
