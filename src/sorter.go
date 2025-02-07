package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/Robogera/detect/pkg/config"
	"github.com/Robogera/detect/pkg/gheap"
	"github.com/Robogera/detect/pkg/indexed"
)

func sorter(
	ctx context.Context, logger *slog.Logger, cfg *config.ConfigFile,
	unsorted_frames_chan <-chan indexed.Indexed[ProcessedFrame],
	sorted_frames_chan chan<- indexed.Indexed[ProcessedFrame]) error {

	queue := gheap.Heap[indexed.Indexed[ProcessedFrame]]{}
	queue.Init()

	// TODO: move to config
	// OR calculate the moving average of the incomig frametime and adjust
	// the ticker period based on it
	ticker := time.NewTicker(time.Second / 60)

	var expected_frame uint64 = 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Streamreader cancelled by context")
			return context.Canceled
		case frame := <-unsorted_frames_chan:
			if frame.Id() < expected_frame {
				continue
			}
			queue.Push(frame)
		case <-ticker.C:
			if queue.IsEmpty() {
				continue
			}
			if queue.Peek().Id() > expected_frame {			
				continue
			}
			frame := queue.Pop()
			select {
			case <-ctx.Done():
				logger.Info("Streamreader cancelled by context")
				return context.Canceled
			case sorted_frames_chan <- frame:
				expected_frame = frame.Id() + 1
			}
		}
	}
}
