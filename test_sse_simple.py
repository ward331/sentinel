#!/usr/bin/env python3
import json
import time
import threading
import requests
import sseclient
from datetime import datetime

print("=== SENTINEL SSE Stream Test ===")
print(f"Test started at: {datetime.now().isoformat()}")
print()

# Global variable to track received events
events_received = []
event_received = threading.Event()

def listen_to_sse():
    """Listen to SSE stream in a separate thread"""
    global events_received
    
    print("Connecting to SSE stream...")
    try:
        # Connect to SSE stream
        response = requests.get('http://localhost:8080/api/events/stream', stream=True)
        response.raise_for_status()
        
        client = sseclient.SSEClient(response)
        
        print("✅ Connected to SSE stream")
        print("Listening for events...")
        print()
        
        for event in client.events():
            if event.data:
                try:
                    data = json.loads(event.data)
                    events_received.append(data)
                    print(f"📨 Received event: {data.get('title', 'Untitled')}")
                    print(f"   ID: {data.get('id')}")
                    print(f"   Source: {data.get('source')}")
                    print(f"   Time: {data.get('occurred_at')}")
                    print()
                    event_received.set()  # Signal that we received an event
                except json.JSONDecodeError as e:
                    print(f"Error parsing event data: {e}")
                    
    except Exception as e:
        print(f"❌ Error in SSE listener: {e}")

def create_test_event():
    """Create a test event to trigger SSE broadcast"""
    time.sleep(2)  # Give SSE listener time to connect
    
    print("Creating test event...")
    
    test_event = {
        "title": f"Python SSE Test {datetime.now().strftime('%H:%M:%S')}",
        "description": "Testing SSE broadcast from Python script",
        "source": "python-test",
        "source_id": f"python-{int(time.time())}",
        "occurred_at": datetime.utcnow().isoformat() + "Z",
        "location": {
            "type": "Point",
            "coordinates": [0, 0]
        },
        "precision": "exact",
        "magnitude": 4.0,
        "category": "earthquake",
        "severity": "medium",
        "metadata": {
            "test": "true",
            "script": "test_sse_simple.py"
        }
    }
    
    try:
        response = requests.post(
            'http://localhost:8080/api/events',
            json=test_event,
            headers={'Content-Type': 'application/json'}
        )
        
        if response.status_code == 201:
            event_data = response.json()
            print(f"✅ Event created: {event_data.get('id')}")
            print(f"   Title: {event_data.get('title')}")
            return event_data.get('id')
        else:
            print(f"❌ Failed to create event: {response.status_code}")
            print(f"   Response: {response.text}")
            return None
            
    except Exception as e:
        print(f"❌ Error creating event: {e}")
        return None

def main():
    # Start SSE listener in a separate thread
    listener_thread = threading.Thread(target=listen_to_sse, daemon=True)
    listener_thread.start()
    
    # Wait a bit for connection to establish
    time.sleep(1)
    
    # Create test event
    event_id = create_test_event()
    
    if event_id:
        print(f"\nWaiting for SSE broadcast of event {event_id}...")
        print("(Timeout: 10 seconds)")
        
        # Wait for event to be received via SSE
        if event_received.wait(timeout=10):
            print("\n✅ SUCCESS: Event received via SSE stream!")
            print(f"Total events received: {len(events_received)}")
            
            # Verify we received the correct event
            for ev in events_received:
                if ev.get('id') == event_id:
                    print(f"✅ Verified: Received the exact event we created")
                    break
        else:
            print("\n❌ TIMEOUT: No events received via SSE within 10 seconds")
            print("Possible issues:")
            print("1. SSE stream might not be broadcasting")
            print("2. Backend might not be calling Broadcast()")
            print("3. Check backend logs for errors")
    
    print(f"\nTest completed at: {datetime.now().isoformat()}")
    print(f"Total events received during test: {len(events_received)}")

if __name__ == "__main__":
    main()