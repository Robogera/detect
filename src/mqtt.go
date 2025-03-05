package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/person"
	"github.com/Robogera/detect/pkg/synapse"

	mqtt "github.com/soypat/natiu-mqtt"
)

func mqttclient(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[[]*person.ExportedPerson],
) error {
	logger := parent_logger.With("coroutine", "mqttclient")
	client := mqtt.NewClient(
		mqtt.ClientConfig{
			Decoder: mqtt.DecoderNoAlloc{UserBuffer: make([]byte, 2048)},
			OnPub: func(pubHead mqtt.Header, varPub mqtt.VariablesPublish, r io.Reader) error {
				message, err := io.ReadAll(r)
				if err != nil {
					return err
				}
				logger.Info("Recieved", "header", pubHead.String(), "message", message)
				return nil
			},
		})
    connection, err := net.Dial("tcp", cfg.Mqtt.Address+":"+fmt.Sprintf("%d", cfg.Mqtt.Port))
	if err != nil {
		logger.Error("TCP connection failed",
			"host", cfg.Mqtt.Address, "port", cfg.Mqtt.Port, "error", err)
		return err
	}

	connection_ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
  var username_bytes, password_bytes []byte
  if cfg.Mqtt.Username != "" {
    username_bytes = []byte(cfg.Mqtt.Username)
  }
  if cfg.Mqtt.Password != "" {
    password_bytes = []byte(cfg.Mqtt.Password)
  }
	vars := &mqtt.VariablesConnect{
		ClientID: []byte(cfg.Mqtt.ClientID),
    Protocol: []byte("mqtt"),
		Username: username_bytes,
		Password: password_bytes,
	}
	err = client.Connect(connection_ctx, connection, vars)
	if err != nil {
		logger.Error("MQTT connection failed",
			"host", cfg.Mqtt.Address, "port", cfg.Mqtt.Port, "client_id", cfg.Mqtt.ClientID,
			"username", cfg.Mqtt.Username, "error", err)
		return err
	}
	logger.Info("MQTT connection established",
		"host", cfg.Mqtt.Address, "port", cfg.Mqtt.Port, "client_id", cfg.Mqtt.ClientID,
		"username", cfg.Mqtt.Username)
	qos := mqtt.QoS0
	pub_flags, err := mqtt.NewPublishFlags(qos, false, false)
	if err != nil {
		logger.Error("MQTT publish flags configured incorrectly", "error", err)
		return err
	}
	base_command := &synapse.Command{
		Sender:    cfg.Mqtt.ClientID,
		Type:      "command",
		Initiator: cfg.Mqtt.ClientID,
		Subject:   "update",
	}
	base_vars := mqtt.VariablesPublish{
		TopicName: []byte(cfg.Mqtt.TopicName),
	}
	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context")
			return context.Canceled
		case frame := <-in_chan:
			base_vars.PacketIdentifier = uint16(frame.Id()+1)
			base_command.Message = &synapse.Message{People: frame.Value()}
			base_command.Id = uint(frame.Id())
			payload, err := base_command.ToPayload()
			if err != nil {
				logger.Error("Can't marshal payload", "frame_id", frame.Id(), "message", frame.Value(), "error", err)
				return err
			}
			err = client.PublishPayload(pub_flags, base_vars, payload)
			if err != nil {
				logger.Error("Can't publish", "frame_id", frame.Id(), "payload", string(payload), "error", err)
				return err
			}
		}
	}
}
