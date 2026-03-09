package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := "3000"
	dir := "."
	
	// Create file server
	fs := http.FileServer(http.Dir(dir))
	
	// Handle CesiumJS assets (they need proper MIME types)
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for SSE
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			return
		}
		
		// If root path, serve the dashboard
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(dir, "sentinel_dashboard.html"))
			return
		}
		
		// Serve the file
		fs.ServeHTTP(w, r)
	}))
	
	fmt.Printf("🌍 SENTINEL Dashboard starting on http://localhost:%s\n", port)
	fmt.Printf("📄 Serving directory: %s\n", dir)
	fmt.Printf("🎯 Dashboard: http://localhost:%s/\n", port)
	fmt.Printf("📊 Test pages:\n")
	fmt.Printf("   • http://localhost:%s/simple_test.html\n", port)
	fmt.Printf("   • http://localhost:%s/cesium_test.html\n", port)
	fmt.Println("🔗 Backend SSE stream: http://localhost:8080/api/events/stream")
	fmt.Println("\nPress Ctrl+C to stop")
	
	// Check if dashboard file exists
	if _, err := os.Stat("sentinel_dashboard.html"); os.IsNotExist(err) {
		fmt.Println("⚠️  WARNING: sentinel_dashboard.html not found!")
		fmt.Println("   Creating default dashboard...")
		createDefaultDashboard()
	}
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func createDefaultDashboard() {
	// Create a simple default dashboard if main one doesn't exist
	html := `<!DOCTYPE html>
<html>
<head>
    <title>SENTINEL Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #0a0e17; color: white; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #667eea; }
        .card { background: rgba(255,255,255,0.05); padding: 20px; border-radius: 10px; margin: 20px 0; }
        a { color: #667eea; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🌍 SENTINEL Dashboard</h1>
        <div class="card">
            <h2>Dashboard Components</h2>
            <p>The main SENTINEL dashboard is being built...</p>
            <ul>
                <li><a href="/simple_test.html">Simple Test Page</a> - Basic SSE testing</li>
                <li><a href="/cesium_test.html">CesiumJS Test Page</a> - 3D globe (requires token)</li>
                <li><a href="http://localhost:8080/api/health">Backend Health</a> - Check backend status</li>
                <li><a href="http://localhost:8080/api/events">Events API</a> - List all events</li>
            </ul>
        </div>
        <div class="card">
            <h2>Real-time Features</h2>
            <p>Backend provides real-time event streaming via Server-Sent Events (SSE).</p>
            <p>SSE Endpoint: <code>http://localhost:8080/api/events/stream</code></p>
        </div>
    </div>
</body>
</html>`
	
	os.WriteFile("default_dashboard.html", []byte(html), 0644)
	fmt.Println("✅ Created default_dashboard.html")
}