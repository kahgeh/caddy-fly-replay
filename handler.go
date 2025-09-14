package flyreplay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// ResponseRecorder captures the response from the upstream
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	header     http.Header
}

// NewResponseRecorder creates a new ResponseRecorder
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		body:          new(bytes.Buffer),
		header:        make(http.Header),
		statusCode:    http.StatusOK,
	}
}

// Header returns the header map
func (r *ResponseRecorder) Header() http.Header {
	return r.header
}

// Write writes the response body
func (r *ResponseRecorder) Write(p []byte) (int, error) {
	return r.body.Write(p)
}

// WriteHeader writes the status code
func (r *ResponseRecorder) WriteHeader(code int) {
	r.statusCode = code
}

// WriteResponse writes the captured response to the original ResponseWriter
func (r *ResponseRecorder) WriteResponse() error {
	// Copy headers from recorder to original response
	for key, values := range r.header {
		for _, value := range values {
			r.ResponseWriter.Header().Add(key, value)
		}
	}
	
	// Write status code
	r.ResponseWriter.WriteHeader(r.statusCode)
	
	// Write body
	_, err := r.ResponseWriter.Write(r.body.Bytes())
	return err
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (f *FlyReplay) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	fullPath := r.Host + r.URL.Path
	
	// Buffer the request body for potential replay
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}
	
	// Step 1: Check cache
	if f.EnableCache && f.cache != nil {
		if cached, found := f.cache.Get(fullPath); found {
			if f.Debug {
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("X-Cached-App", cached.Target)
			}
			
			// Restore body for forwarding to cached app
			if bodyBytes != nil {
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
			
			// Forward directly to cached app
			if app, ok := f.Apps[cached.Target]; ok {
				return f.forwardToApp(w, r, app.Domain)
			}
		} else if f.Debug {
			w.Header().Set("X-Cache", "MISS")
		}
	}
	
	// Restore body for platform
	if bodyBytes != nil {
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	
	// Step 2: Ask platform for routing decision
	rec := NewResponseRecorder(w)
	err := next.ServeHTTP(rec, r)
	if err != nil {
		return err
	}
	
	// Step 3: Check for replay instruction
	if replayHeader := rec.Header().Get("fly-replay"); replayHeader != "" {
		appName := parseAppName(replayHeader)
		
		// Check for cache instruction
		if f.EnableCache && f.cache != nil {
			if cachePattern := rec.Header().Get("fly-replay-cache"); cachePattern != "" {
				if cachePattern == "invalidate" {
					// Platform wants to invalidate cache
					f.cache.Invalidate(fullPath)
					if f.Debug {
						w.Header().Set("X-Cache-Action", "INVALIDATED")
					}
				} else {
					// Platform wants to cache this routing decision
					ttl := f.CacheTTL // default
					if ttlHeader := rec.Header().Get("fly-replay-cache-ttl-secs"); ttlHeader != "" {
						if parsed, err := strconv.Atoi(ttlHeader); err == nil && parsed >= 10 {
							ttl = parsed
						}
					}
					
					// Cache: pattern -> app mapping
					cacheKey := r.Host + cachePattern
					f.cache.Set(fullPath, cacheKey, appName, ttl)
					
					if f.Debug {
						w.Header().Set("X-Cache-Action", "STORED")
						w.Header().Set("X-Cache-Pattern", cacheKey)
					}
				}
			}
		}
		
		// Preserve trace ID from platform response if present
		if traceID := rec.Header().Get("X-Trace-ID"); traceID != "" {
			r.Header.Set("X-Trace-ID", traceID)
		}
		
		// Restore body for forwarding to app
		if bodyBytes != nil {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		
		// Forward to the app
		if app, ok := f.Apps[appName]; ok {
			return f.forwardToApp(w, r, app.Domain)
		}
		
		// App not configured, return error
		return fmt.Errorf("unknown app: %s", appName)
	}
	
	// No replay, return platform's response
	return rec.WriteResponse()
}

// parseAppName extracts the app name from the fly-replay header
func parseAppName(header string) string {
	// Header format: "app=name" or "app=name;instance=xyz"
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "app=") {
			return strings.TrimPrefix(part, "app=")
		}
	}
	return ""
}

// forwardToApp proxies the request to the target app
func (f *FlyReplay) forwardToApp(w http.ResponseWriter, r *http.Request, targetDomain string) error {
	// Parse target URL
	if !strings.HasPrefix(targetDomain, "http://") && !strings.HasPrefix(targetDomain, "https://") {
		targetDomain = "http://" + targetDomain
	}
	
	target, err := url.Parse(targetDomain)
	if err != nil {
		return fmt.Errorf("invalid target domain: %w", err)
	}
	
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Add debug headers if enabled
	if f.Debug {
		w.Header().Set("X-Forwarded-To", targetDomain)
	}
	
	// Serve the request
	proxy.ServeHTTP(w, r)
	return nil
}