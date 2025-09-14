package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	proxyURL = "http://localhost:3000"
	red      = "\033[0;31m"
	green    = "\033[0;32m"
	yellow   = "\033[1;33m"
	blue     = "\033[0;34m"
	cyan     = "\033[0;36m"
	reset    = "\033[0m"
)

type TestCase struct {
	Name        string
	Method      string
	Path        string
	Headers     map[string]string
	Body        interface{}
	ExpectCache string // "miss" or "hit"
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       map[string]interface{}
	RawBody    string
}

func makeRequest(tc TestCase) (*Response, error) {
	var bodyReader io.Reader
	if tc.Body != nil {
		jsonBody, err := json.Marshal(tc.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(tc.Method, proxyURL+tc.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range tc.Headers {
		req.Header.Set(key, value)
	}

	// Set content type for POST/PUT
	if tc.Body != nil && (tc.Method == "POST" || tc.Method == "PUT") {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		RawBody:    string(bodyBytes),
	}

	// Try to parse JSON response
	var jsonBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		response.Body = jsonBody
	}

	return response, nil
}

func printTestHeader(name string) {
	fmt.Printf("\n%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", yellow, reset)
	fmt.Printf("%sTest: %s%s\n", yellow, name, reset)
}

func printRequest(tc TestCase) {
	fmt.Printf("%sRequest:%s %s %s%s\n", cyan, reset, tc.Method, proxyURL+tc.Path, reset)
	if len(tc.Headers) > 0 {
		fmt.Printf("%sHeaders sent:%s\n", cyan, reset)
		for key, value := range tc.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	if tc.Body != nil {
		fmt.Printf("%sBody sent:%s\n", cyan, reset)
		jsonBody, _ := json.MarshalIndent(tc.Body, "  ", "  ")
		fmt.Printf("  %s\n", string(jsonBody))
	}
}

func printResponse(resp *Response, tc TestCase) {
	fmt.Printf("\n%sResponse Status:%s %d\n", green, reset, resp.StatusCode)
	
	// Print relevant headers
	fmt.Printf("%sResponse Headers:%s\n", green, reset)
	relevantHeaders := []string{
		"X-Cache", "X-Cached-App", "X-Cache-Action", "X-Cache-Pattern",
		"X-App-Name", "X-User-ID", "X-Trace-ID", "X-Timestamp",
	}
	
	for _, header := range relevantHeaders {
		if value := resp.Headers.Get(header); value != "" {
			fmt.Printf("  %s: %s\n", header, value)
		}
	}
	
	// Check for echoed headers
	for key := range tc.Headers {
		echoKey := "Echo-" + key
		if value := resp.Headers.Get(echoKey); value != "" {
			fmt.Printf("  %s: %s (echoed)\n", echoKey, value)
		}
	}
	
	// Print response body
	if resp.Body != nil {
		fmt.Printf("%sResponse Body:%s\n", green, reset)
		jsonBody, _ := json.MarshalIndent(resp.Body, "  ", "  ")
		fmt.Printf("  %s\n", string(jsonBody))
	}
}

func verifyTest(resp *Response, tc TestCase) bool {
	passed := true
	fmt.Printf("\n%sVerification:%s\n", blue, reset)
	
	// Check cache behavior
	cacheHeader := resp.Headers.Get("X-Cache")
	if tc.ExpectCache != "" {
		expectedCache := strings.ToUpper(tc.ExpectCache)
		if cacheHeader == expectedCache {
			fmt.Printf("  ✓ Cache behavior correct: %s\n", cacheHeader)
		} else {
			fmt.Printf("  %s✗ Cache behavior incorrect: expected %s, got %s%s\n", red, expectedCache, cacheHeader, reset)
			passed = false
		}
	}
	
	// Check if custom headers were echoed
	for key := range tc.Headers {
		if strings.HasPrefix(key, "X-") {
			echoKey := "Echo-" + key
			if resp.Headers.Get(echoKey) != "" {
				fmt.Printf("  ✓ Header %s was echoed\n", key)
			} else {
				fmt.Printf("  %s✗ Header %s was not echoed%s\n", red, key, reset)
				passed = false
			}
		}
	}
	
	// Check if request body was received (for POST/PUT)
	if tc.Body != nil && (tc.Method == "POST" || tc.Method == "PUT") {
		if resp.Body != nil {
			if received, ok := resp.Body["received"].(map[string]interface{}); ok {
				if receivedBody, ok := received["body"].(string); ok && receivedBody != "" {
					fmt.Printf("  ✓ Request body was received by app\n")
				} else {
					fmt.Printf("  %s✗ Request body was not received by app%s\n", red, reset)
					passed = false
				}
			}
		}
	}
	
	// Check trace ID presence
	if traceID := resp.Headers.Get("X-Trace-ID"); traceID != "" {
		fmt.Printf("  ✓ Trace ID present: %s\n", traceID)
	} else {
		fmt.Printf("  %s✗ Trace ID missing%s\n", red, reset)
		passed = false
	}
	
	return passed
}

func runTests() {
	fmt.Printf("%s=== Comprehensive Caddy Fly-Replay Plugin Test ===%s\n", blue, reset)
	fmt.Printf("Testing header forwarding, body handling, and caching behavior\n")
	
	// Wait for services to be ready
	time.Sleep(2 * time.Second)
	
	testCases := []TestCase{
		{
			Name:   "GET with custom headers - First request (cache miss)",
			Method: "GET",
			Path:   "/en-US/user123/api/profile",
			Headers: map[string]string{
				"X-Custom-Header": "test-value-123",
				"X-Request-ID":    fmt.Sprintf("req-%d", time.Now().Unix()),
				"Authorization":   "Bearer test-token-abc",
				"User-Agent":      "Test-Client/1.0",
			},
			ExpectCache: "miss",
		},
		{
			Name:   "GET with custom headers - Second request (cache hit)",
			Method: "GET",
			Path:   "/en-US/user123/api/settings",
			Headers: map[string]string{
				"X-Custom-Header": "different-value-456",
				"X-Request-ID":    fmt.Sprintf("req-%d", time.Now().Unix()+1),
				"Authorization":   "Bearer different-token-xyz",
			},
			ExpectCache: "hit",
		},
		{
			Name:   "POST with JSON body - First request (cache miss)",
			Method: "POST",
			Path:   "/en-US/user456/api/data",
			Headers: map[string]string{
				"X-Operation": "create",
				"X-Client-ID": "client-789",
			},
			Body: map[string]interface{}{
				"action": "create",
				"data": map[string]interface{}{
					"name":  "Test Item",
					"value": 42,
					"tags":  []string{"test", "demo"},
				},
			},
			ExpectCache: "miss",
		},
		{
			Name:   "POST with JSON body - Second request (cache hit)",
			Method: "POST",
			Path:   "/en-US/user456/api/update",
			Headers: map[string]string{
				"X-Operation": "update",
				"X-Client-ID": "client-999",
			},
			Body: map[string]interface{}{
				"action": "update",
				"id":     123,
				"changes": map[string]interface{}{
					"status":   "active",
					"modified": time.Now().Format(time.RFC3339),
				},
			},
			ExpectCache: "hit",
		},
		{
			Name:   "PUT with large body",
			Method: "PUT",
			Path:   "/en-US/user789/api/resource/1",
			Headers: map[string]string{
				"X-Resource-Type": "document",
				"X-Version":       "2.0",
			},
			Body: map[string]interface{}{
				"id":          1,
				"title":       "Resource Title",
				"description": "A longer description that contains more text to verify body forwarding works correctly even with larger payloads. This should be properly forwarded through the proxy chain.",
				"metadata": map[string]interface{}{
					"created":  "2024-01-01",
					"tags":     []string{"test", "demo", "verification", "proxy", "forwarding"},
					"priority": "high",
					"attributes": map[string]string{
						"type":     "document",
						"category": "testing",
						"owner":    "test-user",
					},
				},
			},
			ExpectCache: "miss",
		},
		{
			Name:   "Headers with special characters",
			Method: "GET",
			Path:   "/en-US/user123/api/special",
			Headers: map[string]string{
				"X-Special-Chars": "Hello, World! @#$%^&*()",
				"X-Base64":        "SGVsbG8gV29ybGQ=",
				"X-Url-Encoded":   "value%20with%20spaces",
			},
			ExpectCache: "hit", // Should hit cache from earlier user123 requests
		},
	}
	
	totalTests := len(testCases)
	passedTests := 0
	
	for i, tc := range testCases {
		printTestHeader(fmt.Sprintf("[%d/%d] %s", i+1, totalTests, tc.Name))
		printRequest(tc)
		
		resp, err := makeRequest(tc)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", red, err, reset)
			continue
		}
		
		printResponse(resp, tc)
		
		if verifyTest(resp, tc) {
			passedTests++
			fmt.Printf("%s✓ Test passed%s\n", green, reset)
		} else {
			fmt.Printf("%s✗ Test failed%s\n", red, reset)
		}
		
		// Small delay between tests
		time.Sleep(500 * time.Millisecond)
	}
	
	// Print summary
	fmt.Printf("\n%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", blue, reset)
	fmt.Printf("%s=== Test Summary ===%s\n", blue, reset)
	fmt.Printf("Tests passed: %d/%d\n", passedTests, totalTests)
	
	if passedTests == totalTests {
		fmt.Printf("%s✓ All tests passed!%s\n", green, reset)
	} else {
		fmt.Printf("%s✗ Some tests failed%s\n", red, reset)
	}
	
	fmt.Printf("\n%sKey verifications:%s\n", cyan, reset)
	fmt.Println("• Custom headers are forwarded through the proxy chain")
	fmt.Println("• Request bodies are preserved for POST/PUT requests")
	fmt.Println("• Response bodies are returned correctly")
	fmt.Println("• Trace IDs are maintained across requests")
	fmt.Println("• Cache behavior works with different HTTP methods")
	fmt.Println("• Special characters in headers are handled properly")
}

func main() {
	// Check if services are running
	fmt.Printf("%sChecking if services are running...%s\n", yellow, reset)
	
	_, err := http.Get("http://localhost:3000")
	if err != nil {
		log.Fatalf("%sCaddy is not running on port 3000. Please start services first.%s\n", red, reset)
	}
	
	runTests()
}