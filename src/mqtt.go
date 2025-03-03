package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/indexed"
	"github.com/Robogera/detect/pkg/person"

	mqtt "github.com/soypat/natiu-mqtt"
)

func mqttclient(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile,
	in_chan <-chan indexed.Indexed[[]*person.Person],
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
		connection, err := net.Dial("tcp", "127.0.0.1:1883")
		if err != nil {
			return err
		}

		connection_ctx, cancel := context.WithTimeout(ctx, time.Second * 5)
		defer cancel()
		vars := &mqtt.VariablesConnect{
			ClientID: []byte("tracker"),
			Username: []byte("tracker"),
			Password: []byte("tracker"),
		}
		err = client.Connect(connection_ctx, connection, &mqtt.VariablesConnect{})
		if err != nil {
			return err
		}
	return nil
}
