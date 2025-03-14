package config

import (
	// stdlib
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
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
	Mqtt      MqttConfig
	Mask      MaskConfig
	Crop      CropConfig
}

type CropConfig struct {
	A Point
	B Point
}

type MaskConfig struct {
	Contours []Contour
	Color    Color
}

type Color struct {
	R, G, B uint8
}

type Contour []Point

type Point struct {
	X, Y uint
}

type MqttConfig struct {
	Address   string `toml:"address"`
	Port      uint   `toml:"port" comment:"usually 1883"`
	TopicName string `toml:"topic_name"`
	ClientID  string `toml:"client_id"`
	Username  string `toml:"username"`
	Password  string `toml:"password"`
	Type      string `toml:"type"`
	Subject   string `toml:"subject"`
}

type ReidConfig struct {
	Format            string  `toml:"format" comment:"onnx, openvino or caffe"`
	Path              string  `toml:"path"`
	ConfigPath        string  `toml:"config_path" comment:"required for caffe models"`
	OutputLayerName   string  `toml:"output_layer_name" comment:"usually 'reid_embedding'"`
	SMAWindow         uint    `toml:"sma_window" comment:"higher values for smoother trajectory at the cost of higher delay"`
	TotalDescriptors  uint    `toml:"total_descriptors" comment:"higher values improve reidentification at the cost of performance"`
	PredictSec        float64 `toml:"predict_sec" comment:"stops trying to predict the person's movement after specified time"`
	ValidateSec       float64 `toml:"validate_sec" comment:"higher values filter out false positives at the cost of higher delay when discovering new people"`
	ExpireSec         float64 `toml:"expire_sec" comment:"expire tracked people after specified time"`
	NonValidExpireSec float64 `toml:"nonvalid_expire_sec" comment:"expire unvalidated people after specified time"`
	ValidationFrames  uint    `toml:"validation_frames" comment:"minimum frames to detect before validation_duration to validate"`
	ScoreThreshold    float64 `toml:"score_threshold" comment:"minimum score to associate people"`
	DistanceFactor    float64 `toml:"distance_factor" comment:"divide distances above threshold"`
	DistanceThreshold uint    `toml:"distance_threshold" comment:"distance after which the factor is applied to score"`
	TokenLength       uint    `toml:"token_length" comment:"only affects log readability really"`
}

type KalmanConfig struct {
	ProcessNoiseCov float64 `toml:"process_noise_cov"`
	MeasNoiseCov    float64 `toml:"measurement_noise_cov"`
}

type YoloConfig struct {
	Format              string  `toml:"format" comment:"onnx, openvino or caffe"`
	Path                string  `toml:"path"`
	ConfigPath          string  `toml:"config_path" comment:"required for caffe models"`
	Transpose           bool    `toml:"transpose" comment:"set true for ultralythics-authored models"`
	ScaleFactor         float64 `toml:"scale_factor"`
	W                   uint    `toml:"w"`
	H                   uint    `toml:"h"`
	ConfidenceThreshold float32 `toml:"confidence_threshold"`
	NMSThreshold        float32 `toml:"nms_threshold" comment:"lower values for more aggressive filtering"`
	PersonClassIndex    uint    `toml:"person_class_index" comment:"0 or 1 for the majority of pre-trained models"`
	Threads             uint    `toml:"threads" comment:"higher values increase performance on multicore systems"`
	SortingFPS          uint    `toml:"sorting_fps" comment:"frequency of post-yolo sorter output"`
}

type BackendConfig struct {
	Device string `toml:"device" comment:"cpu, vpu or gpu"`
}

type InputConfig struct {
	Type string `toml:"type" comment:"file, stream or webcam"`
	Path string `toml:"path" comment:"for file or stream types"`
}

type WebserverConfig struct {
	Port               uint `toml:"port"`
	ReadTimeoutSec     uint `toml:"read_timeout_sec" comment:"0 for no timeout"`
	WriteTimeoutSec    uint `toml:"write_timeout_sec" comment:"0 fot no timeout"`
	ShutdownTimeoutSec uint `toml:"shutdown_timeout_sec"`
	W                  uint `toml:"width" comment:"if either is zero - no resizing will be done"`
	H                  uint `toml:"height" comment:"if either is zero - no resizing will be done"`
}

type LoggingConfig struct {
	Level         string `toml:"level" comment:"debug, info, warn or error"`
	StatPeriodSec uint   `toml:"stat_period_sec"`
}

func Migrate(file_path string) error {
	config_file, err := Unmarshal(file_path)
	if err != nil {
		return err
	}
	new_path, err := getLegalIncrementedFileName(file_path)
	if err != nil {
		return err
	}
	os.Rename(file_path, new_path)
	err = Write2File(config_file, file_path)
	if err != nil {
		return err
	}
	return nil
}

func getLegalIncrementedFileName(file_path string) (string, error) {
	for i := range int(^uint(0) >> 1) {
		name := fmt.Sprintf("%s.%d", file_path, i)
		if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
			return name, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("fuck you")
}

func CreateDefault(file_path string) error {
	config_file := new(ConfigFile)
	config_file.Reid = ReidConfig{
		Format:            "onnx",
		Path:              "/my/model.onnx",
		ConfigPath:        "/my/config.xml",
		OutputLayerName:   "reid_embedding",
		SMAWindow:         5,
		TotalDescriptors:  3,
		PredictSec:        0.3,
		ValidateSec:       1.0,
		ExpireSec:         2.0,
		DistanceFactor:    2,
		DistanceThreshold: 150,
		ScoreThreshold:    0.001,
		ValidationFrames:  5,
		TokenLength:       4,
	}
	config_file.Yolo = YoloConfig{
		Format:              "onnx",
		Path:                "/my/model.onnx",
		ConfigPath:          "/my/config.xml",
		Transpose:           false,
		ScaleFactor:         255.0,
		W:                   640,
		H:                   480,
		ConfidenceThreshold: 0.995,
		NMSThreshold:        0.05,
		PersonClassIndex:    1,
		Threads:             3,
	}
	config_file.Kalman = KalmanConfig{
		ProcessNoiseCov: 0.01,
		MeasNoiseCov:    600,
	}
	config_file.Backend = BackendConfig{
		Device: "cpu",
	}
	config_file.Input = InputConfig{
		Type: "stream",
		Path: "rtsc://myweb.cam:544/Stream/111",
	}
	config_file.Webserver = WebserverConfig{
		Port:               8080,
		ReadTimeoutSec:     0,
		WriteTimeoutSec:    0,
		ShutdownTimeoutSec: 3,
	}
	config_file.Logging = LoggingConfig{
		Level:         "info",
		StatPeriodSec: 4,
	}
	config_file.Mqtt = MqttConfig{
		Address:   "127.0.0.1",
		Port:      1883,
		TopicName: "tracking",
		ClientID:  "01",
		Username:  "user",
		Password:  "pass",
	}
	return Write2File(config_file, file_path)
}

func Write2File(config_file *ConfigFile, file_path string) error {
	file, err := os.Create(file_path)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("Can't create %s: %w", file_path, err)
	}
	data, err := toml.Marshal(config_file)
	if err != nil {
		return fmt.Errorf("Can't serialize default config: %w", err)
	}
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("Can't write to %s: %w", file_path, err)
	}
	return nil
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
