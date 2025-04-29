// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type EventLoop struct {
	handler EventHandler
	eventCh <-chan any
	logger  slog.Logger

	currentBatch EventBatch
	nextBatch    EventBatch

	handling bool
	mu       sync.Mutex
}

// NewEventLoop creates a new EventLoop.
func NewEventLoop(
	eventCh <-chan any,
	logger slog.Logger,
	handler EventHandler,
) *EventLoop {
	return &EventLoop{
		eventCh:      eventCh,
		logger:       logger,
		handler:      handler,
		currentBatch: EventBatch{Events: make([]any, 0), BatchID: 0},
		nextBatch:    EventBatch{Events: make([]any, 0), BatchID: 1},
	}
}

func (*EventLoop) NeedLeaderElection() bool {
	// Leader election (= be leader) is not required for this loop to start.
	// false = always run even if not leader
	return false
}

// Start starts the EventLoop.
// This method will block until the EventLoop stops, which will happen after the ctx is closed.
func (el *EventLoop) Start(ctx context.Context) error {
	// handlingDone is used to signal the completion of handling a batch.
	handlingDone := make(chan struct{})

	handleBatch := func() {
		go func(batch EventBatch) {
			el.SetHandling(true)
			// batchLogger := el.logger.WithName("batchHandler").WithValues("batchID", el.currentBatch.BatchID)
			batchLogger := el.logger.WithGroup("batchHandler").With("batchID", el.currentBatch.BatchID)
			batchLogger.Info("Handling events from the batch", "total", len(batch.Events))

			el.handler.HandleEventBatch(ctx, batchLogger, batch)

			batchLogger.Info("... Sleeping, give it some time to get more events in the next batch")
			time.Sleep(5 * time.Second)
			batchLogger.Info("... Slept")

			batchLogger.Info("Finished handling the batch")
			handlingDone <- struct{}{}
		}(el.currentBatch)
	}

	swapAndHandleBatch := func() {
		el.swapBatches()
		handleBatch()
	}

	// The event monitoring loop
	for {
		select {
		case <-ctx.Done():
			// Wait for the completion if a batch is being handled.
			if el.GetHandling() {
				<-handlingDone
			}
			return nil
		case e := <-el.eventCh:
			// Add the event to the current batch.
			el.nextBatch.Events = append(el.nextBatch.Events, e)

			el.logger.Info(
				"added an event to the next batch",
				"batchId", el.nextBatch.BatchID,
				"type", fmt.Sprintf("%T", e),
				"event", e,
				"total", len(el.nextBatch.Events),
			)
			// If no batch is currently being handled, swap batches and begin handling the batch.
			if !el.GetHandling() {
				swapAndHandleBatch()
			}
		case <-handlingDone:
			el.SetHandling(false)

			// If there's at least one event in the next batch, swap batches and begin handling the batch.
			if len(el.nextBatch.Events) > 0 {
				swapAndHandleBatch()
			}
		}
	}
}

// swapBatches swaps the current and next batches.
func (el *EventLoop) swapBatches() {
	el.currentBatch, el.nextBatch = el.nextBatch, el.currentBatch
	el.nextBatch.Events = el.nextBatch.Events[:0]
	el.nextBatch.BatchID = el.currentBatch.BatchID + 1
}

func (el *EventLoop) SetHandling(value bool) {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.handling = value
}

func (el *EventLoop) GetHandling() bool {
	el.mu.Lock()
	defer el.mu.Unlock()
	return el.handling
}
