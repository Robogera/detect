package main

import (
	// stdlib
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	// internal
	"github.com/Robogera/detect/pkg/config"

	// external
	"github.com/hybridgroup/mjpeg"
)

func webplayer(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ConfigFile, // wish I could pass this as read only to prevent subroutines messing the configuration or data races...
	frames_chan <-chan []byte,
) error {

	output_stream := mjpeg.NewStream()

	http.Handle("/", output_stream)

	server := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", cfg.Webserver.Port),
		ReadTimeout:  time.Duration(cfg.Webserver.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Webserver.WriteTimeoutSec) * time.Second,
	}

	err_chan := make(chan error)

	go func() {
		err_chan <- server.ListenAndServe()
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Webserver cancelled by context. Shutting down...", "timeout (sec)", cfg.Webserver.ShutdownTimeoutSec)

			shutdown_context, cancel := context.WithTimeout(
				context.Background(),
				time.Second*time.Duration(cfg.Webserver.ShutdownTimeoutSec))
			defer cancel()
			shutdown_initiated_timestamp := time.Now()
			err := server.Shutdown(shutdown_context)
			logger.Info(
				"Webserver shut down successfully", "shutdown duration (sec)",
				time.Now().Sub(shutdown_initiated_timestamp).Seconds(), "error", err)
			return ERR_CANCELLED_BY_CONTEXT
		case err := <-err_chan:
			logger.Error("Server error", "port", cfg.Webserver.Port, "error", err)
			return err
		case frame := <-frames_chan:
			output_stream.UpdateJPEG(frame)
		}
	}
}