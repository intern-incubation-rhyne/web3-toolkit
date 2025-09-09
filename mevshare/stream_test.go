package mevshare_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"toolkit/mevshare"
)

func TestStreamReaderWithRetry(t *testing.T) {
	// Create stream reader
	reader := mevshare.NewStreamReader(mevshare.Mainnet)

	// Create context with timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create event channel
	eventChan := make(chan mevshare.StreamEvent, 100)

	// Start reading stream with retry in goroutine
	go reader.ReadStreamWithRetry(ctx, eventChan, 5*time.Second)

	// Print all events from the channel
	eventCount := 0
	startTime := time.Now()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				t.Log("Event channel closed")
				return
			}

			eventCount++
			elapsed := time.Since(startTime)

			data, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			fmt.Printf("[%s] Event #%d - Hash: %s\n", elapsed.Round(time.Second), eventCount, event.Hash)
			fmt.Println(string(data))
			fmt.Println("--------------------------------")

		case <-ctx.Done():
			fmt.Printf("Context cancelled after %v, received %d events\n", time.Since(startTime), eventCount)
			return
		}
	}
}
