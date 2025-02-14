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
	"github.com/Robogera/detect/pkg/indexed"
	"gocv.io/x/gocv"

	// external
	"github.com/hybridgroup/mjpeg"
)

func webplayer(
	ctx context.Context,
	parent_logger *slog.Logger,
	cfg *config.ConfigFile, // wish I could pass this as read only to prevent subroutines messing the configuration or data races...
	in_chan <-chan indexed.Indexed[ProcessedFrame],
	stat_chan chan<- Statistics,
) error {

	logger := parent_logger.With("coroutine", "webplayer")

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
	defer func() {
		shutdown_context, cancel := context.WithTimeout(
			context.Background(),
			time.Second*time.Duration(cfg.Webserver.ShutdownTimeoutSec))
		defer cancel()
		shutdown_initiated_timestamp := time.Now()
		err := server.Shutdown(shutdown_context)
		logger.Info(
			"Shut down",
			"shutdown time (sec)", time.Now().Sub(shutdown_initiated_timestamp).Seconds(),
			"error", err)
	}()

	logger.Info("Started", "port", cfg.Webserver.Port)

	last_frame_timestamp := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Cancelled by context", "timeout (sec)", cfg.Webserver.ShutdownTimeoutSec)

			return context.Canceled
		case err := <-err_chan:
			logger.Error("Error", "port", cfg.Webserver.Port, "error", err)
			return err
		case frame := <-in_chan:
			buf, err := gocv.IMEncode(gocv.JPEGFileExt, *frame.Value().Mat)
			if err != nil {
				logger.Error("Can't encode frame")
				return err
			}
			data := make([]byte, buf.Len())
			copy(data, buf.GetBytes()) // need to profile this and maybe not copy the entire frame every time
			output_stream.UpdateJPEG(data)
			buf.Close()
			frame.Value().Mat.Close()
			select {
			case stat_chan <- Statistics{time.Since(last_frame_timestamp)}:
				last_frame_timestamp = time.Now()
			case <-ctx.Done():
				logger.Info("Cancelled by context")
				return context.Canceled
			}
		}
	}
}
