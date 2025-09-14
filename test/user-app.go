package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// generateTraceID generates a random trace ID
func generateTraceID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Mock user app server with enhanced header and body handling
func main() {
	var (
		port   = flag.String("port", "9001", "Port to listen on")
		userID = flag.String("user", "user123", "User ID for this app")
	)
	flag.Parse()
	
	appName := fmt.Sprintf("%s-app", *userID)
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check for trace ID in the request
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			newTraceID := generateTraceID()
			log.Printf("[%s] No trace ID received, generated new: %s", appName, newTraceID)
			traceID = newTraceID
		} else {
			log.Printf("[%s] Received trace ID: %s", appName, traceID)
		}
		
		log.Printf("[%s] [TraceID: %s] Received request: %s %s", appName, traceID, r.Method, r.URL.Path)
		
		// Log all received headers
		log.Printf("[%s] [TraceID: %s] Request Headers:", appName, traceID)
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("[%s] [TraceID: %s]   %s: %s", appName, traceID, name, value)
			}
		}
		
		// Read request body if present
		var requestBody string
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil && len(bodyBytes) > 0 {
				requestBody = string(bodyBytes)
				log.Printf("[%s] [TraceID: %s] Request Body: %s", appName, traceID, requestBody)
			}
		}
		
		// Echo back all custom headers (X-* headers)
		for name, values := range r.Header {
			if strings.HasPrefix(name, "X-") {
				for _, value := range values {
					w.Header().Add("Echo-"+name, value)
				}
			}
		}
		
		// Add response headers to identify which app handled the request
		w.Header().Set("X-App-Name", appName)
		w.Header().Set("X-User-ID", *userID)
		w.Header().Set("X-Trace-ID", traceID)
		w.Header().Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
		w.Header().Set("Content-Type", "application/json")
		
		// Build response that includes received data
		response := map[string]interface{}{
			"app":       appName,
			"user":      *userID,
			"traceId":   traceID,
			"path":      r.URL.Path,
			"method":    r.Method,
			"timestamp": time.Now().Unix(),
			"message":   fmt.Sprintf("Hello from %s's application!", *userID),
			"received": map[string]interface{}{
				"headers": func() map[string][]string {
					// Return custom headers only
					customHeaders := make(map[string][]string)
					for name, values := range r.Header {
						if strings.HasPrefix(name, "X-") || 
						   name == "Authorization" || 
						   name == "User-Agent" {
							customHeaders[name] = values
						}
					}
					return customHeaders
				}(),
				"body": requestBody,
			},
		}
		
		// Send JSON response
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			log.Printf("[%s] [TraceID: %s] Error encoding response: %v", appName, traceID, err)
		}
		
		log.Printf("[%s] [TraceID: %s] Response sent successfully", appName, traceID)
	})
	
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("[%s] Starting enhanced user app on port %s", appName, *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}