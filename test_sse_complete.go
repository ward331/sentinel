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
	fmt.Println("=== SENTINEL Backend SSE Stream Test ===")
	fmt.Println("Testing connection to: http://localhost:8080/api/events/stream")
	fmt.Println()
	
	// First, connect to SSE stream
	fmt.Println("1. Connecting to SSE stream...")
	
	req, err := http.NewRequest("GET", "http://localhost:8080/api/events/stream", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting to SSE: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		fmt.Printf("SSE endpoint returned status: %d\n", resp.StatusCode)
		return
	}
	
	fmt.Printf("✅ Connected successfully (Status: %d)\n", resp.StatusCode)
	fmt.Println()
	
	// Start reading SSE events in a goroutine
	eventChan := make(chan string)
	go func() {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("SSE stream closed by server")
				} else {
					fmt.Printf("Error reading SSE: %v\n", err)
				}
				close(eventChan)
				return
			}
			
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(line[5:])
				if data != "" {
					eventChan <- data
				}
			}
		}
	}()
	
	// Create a test event to trigger SSE broadcast
	fmt.Println("2. Creating test event to trigger SSE...")
	
	testEvent := map[string]interface{}{
		"title":       "SSE Test Earthquake",
		"description": "Testing SSE stream broadcast functionality",
		"source":      "test",
		"source_id":   fmt.Sprintf("sse-test-%d", time.Now().Unix()),
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{-122.4194, 37.7749}, // San Francisco
		},
		"precision": "exact",
		"magnitude": 4.2,
		"category":  "earthquake",
		"severity":  "medium",
		"metadata": map[string]string{
			"test":     "true",
			"purpose":  "SSE verification",
			"backend":  "SENTINEL",
		},
	}
	
	eventJSON, _ := json.Marshal(testEvent)
	
	req2, _ := http.NewRequest("POST", "http://localhost:8080/api/events", strings.NewReader(string(eventJSON)))
	req2.Header.Set("Content-Type", "application/json")
	
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("Error creating test event: %v\n", err)
	} else {
		defer resp2.Body.Close()
		if resp2.StatusCode == 201 {
			fmt.Println("✅ Test event created successfully")
		} else {
			body, _ := io.ReadAll(resp2.Body)
			fmt.Printf("Event creation failed (Status: %d): %s\n", resp2.StatusCode, string(body))
		}
	}
	
	fmt.Println()
	fmt.Println("3. Waiting for SSE events (timeout: 10 seconds)...")
	fmt.Println()
	
	// Wait for events with timeout
	timeout := time.After(10 * time.Second)
	eventCount := 0
	
	for {
		select {
		case eventData, ok := <-eventChan:
			if !ok {
				fmt.Println("SSE channel closed")
				return
			}
			
			eventCount++
			fmt.Printf("📨 Received Event #%d:\n", eventCount)
			
			// Parse and display the event
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(eventData), &event); err != nil {
				fmt.Printf("   Error parsing JSON: %v\n", err)
				fmt.Printf("   Raw data: %s\n", eventData)
			} else {
				if title, ok := event["title"].(string); ok {
					fmt.Printf("   Title: %s\n", title)
				}
				if category, ok := event["category"].(string); ok {
					fmt.Printf("   Category: %s\n", category)
				}
				if magnitude, ok := event["magnitude"].(float64); ok {
					fmt.Printf("   Magnitude: %.1f\n", magnitude)
				}
				if source, ok := event["source"].(string); ok {
					fmt.Printf("   Source: %s\n", source)
				}
				if id, ok := event["id"].(string); ok {
					fmt.Printf("   ID: %s\n", id)
				}
			}
			fmt.Println()
			
		case <-timeout:
			if eventCount == 0 {
				fmt.Println("❌ No events received within timeout period")
				fmt.Println("\nPossible issues:")
				fmt.Println("1. SSE stream might not be broadcasting events")
				fmt.Println("2. Backend might need to be restarted")
				fmt.Println("3. Check backend logs for 'Broadcasted new event' messages")
			} else {
				fmt.Printf("✅ Test completed. Received %d event(s)\n", eventCount)
			}
			return
		}
	}
}