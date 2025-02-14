package config

import (
	// stdlib
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
)

// Enum types

type ModelFormat string

const (
	ModelFormatONNX     = "onnx"
	ModelFormatOpenVINO = "openvino"
	ModelFormatCaffe    = "caffe"
)

type LoggingLevel string

const (
	LoggingLevelDebug = "debug"
	LoggingLevelInfo  = "info"
	LoggingLevelWarn  = "warn"
	LoggingLevelError = "error"
)

type DeviceType string

const (
	DeviceTypeCPU = "cpu"
	DeviceTypeVPU = "vpu"
	DeviceTypeGPU = "gpu"
)

type InputType string

const (
	InputTypeFile   = "file"
	InputTypeWebcam = "webcam"
	InputTypeIPC    = "ipc"
)

// Config file structure

type ConfigFile struct {
	Yolo      YoloConfig
	Reid      ReidConfig
	Kalman    KalmanConfig
	Backend   BackendConfig
	Webserver WebserverConfig
	Logging   LoggingConfig
	Input     InputConfig
}

type ReidConfig struct {
	Format          string
	Path            string
	ConfigPath      string  `toml:"config_path"`
	OutputLayerName string  `toml:"output_layer_name"`
	ScoreThreshold  float64 `toml:"score_threshold"`
	SMAWindow       int     `toml:"sma_window"`
	FramesToFollow  int     `toml:"frames_to_follow"`
	TTL             float64 `toml:"time_to_live"`
	DistanceFactor  float64 `toml:"distance_factor"`
}

type KalmanConfig struct {
	ProcessNoiseCov float64 `toml:"process_noise_cov"`
	MeasNoiseCov    float64 `toml:"measurement_noise_cov"`
}

type YoloConfig struct {
	Format              string
	Path                string
	ConfigPath          string `toml:"config_path"`
	Transpose           bool
	ScaleFactor         float64 `toml:"scale_factor"`
	X                   uint
	Y                   uint
	ConfidenceThreshold float32 `toml:"confidence_threshold"`
	NMSThreshold        float32 `toml:"nms_threshold"`
	PersonClassIndex    uint    `toml:"person_class_index"`
	Threads             uint
}

type BackendConfig struct {
	Device string
}

type InputConfig struct {
	Type string
	Path string
}

type WebserverConfig struct {
	Port               uint
	ReadTimeoutSec     uint `toml:"read_timeout_sec"`
	WriteTimeoutSec    uint `toml:"write_timeout_sec"`
	ShutdownTimeoutSec uint `toml:"shutdown_timeout_sec"`
}

type LoggingConfig struct {
	Level         string
	StatPeriodSec uint `toml:"stat_period_sec"`
}

func Unmarshal(file_path string) (*ConfigFile, error) {
	config_file := new(ConfigFile)
	data, err := os.ReadFile(file_path)
	if err != nil {
		return nil,
			fmt.Errorf("Unable to read %s error: %w", file_path, err)
	}
	err = toml.Unmarshal(data, config_file)
	if err != nil {
		return nil,
			fmt.Errorf("Unable to unmarshal %s error: %w", file_path, err)
	}
	return config_file, nil
}
