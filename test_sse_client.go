package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== SENTINEL SSE Client Test ===")
	fmt.Println("Connecting to: http://localhost:8080/api/events/stream")
	fmt.Println()

	// Make HTTP request
	req, err := http.NewRequest("GET", "http://localhost:8080/api/events/stream", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Println()

	if resp.StatusCode != 200 {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		return
	}

	// Read SSE stream
	reader := bufio.NewReader(resp.Body)
	eventCount := 0
	
	fmt.Println("Listening for SSE events (timeout: 15 seconds)...")
	fmt.Println()

	// Set a timeout
	timeout := time.After(15 * time.Second)
	
	for {
		select {
		case <-timeout:
			fmt.Printf("\nTimeout reached. Received %d event(s).\n", eventCount)
			if eventCount == 0 {
				fmt.Println("No events received. Possible issues:")
				fmt.Println("1. No events being broadcast")
				fmt.Println("2. SSE format might be incorrect")
				fmt.Println("3. Check backend logs for 'Broadcasting event' messages")
			}
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("Server closed connection")
				} else {
					fmt.Printf("Read error: %v\n", err)
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			fmt.Printf("Raw line: %s\n", line)
			
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(line[5:])
				if data != "" {
					eventCount++
					fmt.Printf("\n=== EVENT #%d ===\n", eventCount)
					
					// Try to parse as JSON
					var event map[string]interface{}
					if err := json.Unmarshal([]byte(data), &event); err == nil {
						if id, ok := event["id"].(string); ok {
							fmt.Printf("ID: %s\n", id)
						}
						if title, ok := event["title"].(string); ok {
							fmt.Printf("Title: %s\n", title)
						}
						if source, ok := event["source"].(string); ok {
							fmt.Printf("Source: %s\n", source)
						}
						if category, ok := event["category"].(string); ok {
							fmt.Printf("Category: %s\n", category)
						}
					} else {
						fmt.Printf("Data: %s\n", data)
					}
					fmt.Println()
				}
			} else if strings.HasPrefix(line, "event:") {
				eventType := strings.TrimSpace(line[6:])
				fmt.Printf("Event type: %s\n", eventType)
			} else if strings.HasPrefix(line, ":") {
				// Comment
				comment := strings.TrimSpace(line[1:])
				fmt.Printf("Comment: %s\n", comment)
			}
		}
	}
}