package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// generateTraceID generates a random trace ID
func generateTraceID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Mock platform service that decides routing based on path
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check for existing trace ID or generate new one
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
			log.Printf("[PLATFORM] No trace ID received, generated new: %s", traceID)
		} else {
			log.Printf("[PLATFORM] Received trace ID: %s", traceID)
		}
		
		log.Printf("[PLATFORM] [TraceID: %s] Received request: %s %s from %s", traceID, r.Method, r.URL.Path, r.RemoteAddr)
		
		// Parse the path to determine which user app should handle this
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		
		// Expected format: /locale/userID/...
		if len(parts) >= 2 {
			locale := parts[0]
			userID := parts[1]
			
			// Determine which app based on userID
			var appName string
			switch userID {
			case "user123":
				appName = "user123-app"
			case "user456":
				appName = "user456-app"
			case "user789":
				appName = "user789-app"
			default:
				// Unknown user
				log.Printf("[PLATFORM] [TraceID: %s] Unknown user: %s", traceID, userID)
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			
			// Set fly-replay header to instruct Caddy to route to the app
			w.Header().Set("fly-replay", fmt.Sprintf("app=%s", appName))
			
			// Include trace ID in the response headers
			w.Header().Set("X-Trace-ID", traceID)
			
			// Tell Caddy to cache this routing decision
			cachePattern := fmt.Sprintf("/%s/%s/*", locale, userID)
			w.Header().Set("fly-replay-cache", cachePattern)
			w.Header().Set("fly-replay-cache-ttl-secs", "300") // Cache for 5 minutes
			
			log.Printf("[PLATFORM] [TraceID: %s] Routing to %s, cache pattern: %s%s", traceID, appName, r.Host, cachePattern)
			
			// Return a response (Caddy will intercept and replay)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Platform routing to %s\n", appName)
		} else {
			// Invalid path format
			log.Printf("[PLATFORM] [TraceID: %s] Invalid path format: %s", traceID, r.URL.Path)
			http.Error(w, "Invalid path format", http.StatusBadRequest)
		}
	})
	
	port := ":8080"
	log.Printf("[PLATFORM] Starting platform service on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}