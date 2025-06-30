package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"web-analyzer/internal/config"

	"golang.org/x/net/html"
)

func TestNew(t *testing.T) {
	cfg := config.AnalyzerConfig{
		RequestTimeout: 10 * time.Second,
		LinkTimeout:    5 * time.Second,
		MaxRedirects:   3,
		MaxWorkers:     5,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	analyzer := New(cfg, logger)

	if analyzer == nil {
		t.Fatal("New() returned nil")
	}

	if analyzer.config.RequestTimeout != cfg.RequestTimeout {
		t.Errorf("Expected RequestTimeout %v, got %v", cfg.RequestTimeout, analyzer.config.RequestTimeout)
	}

	if analyzer.client.Timeout != cfg.RequestTimeout {
		t.Errorf("Expected client timeout %v, got %v", cfg.RequestTimeout, analyzer.client.Timeout)
	}

	if analyzer.logger == nil {
		t.Fatal("Logger is nil")
	}
}

func TestAnalyzeURL_CompleteAnalysis(t *testing.T) {
	testHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Web Page Analysis</title>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Main Heading</h1>
    <h2>Section One</h2>
    <h2>Section Two</h2>
    <h3>Subsection</h3>
    
    <nav>
        <a href="/about">About Us</a>
        <a href="/contact">Contact</a>
        <a href="https://external-site.com">External Link</a>
    </nav>
    
    <main>
        <p>Some content here</p>
        <a href="/internal-page">Internal Page</a>
    </main>
    
    <footer>
        <form id="login-form">
            <input type="email" name="email" placeholder="Email">
            <input type="password" name="password" placeholder="Password">
            <button type="submit">Login</button>
        </form>
        
        <form id="search-form">
            <input type="text" name="query" placeholder="Search">
            <button type="submit">Search</button>
        </form>
    </footer>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, testHTML)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	result, err := analyzer.AnalyzeURL(context.Background(), server.URL)

	if err != nil {
		t.Fatalf("AnalyzeURL failed: %v", err)
	}

	// Test title extraction
	if result.Title != "Test Web Page Analysis" {
		t.Errorf("Expected title 'Test Web Page Analysis', got '%s'", result.Title)
	}

	// Test HTML version detection
	if result.HTMLVersion != "HTML5" {
		t.Errorf("Expected HTML5, got %s", result.HTMLVersion)
	}

	// Test heading counts
	if result.Headings["h1"] != 1 {
		t.Errorf("Expected 1 h1, got %d", result.Headings["h1"])
	}
	if result.Headings["h2"] != 2 {
		t.Errorf("Expected 2 h2, got %d", result.Headings["h2"])
	}
	if result.Headings["h3"] != 1 {
		t.Errorf("Expected 1 h3, got %d", result.Headings["h3"])
	}

	// Test link classification
	if result.InternalLinks != 3 {
		t.Errorf("Expected 3 internal links, got %d", result.InternalLinks)
	}
	if result.ExternalLinks != 1 {
		t.Errorf("Expected 1 external link, got %d", result.ExternalLinks)
	}

	// Test login form detection
	if !result.HasLoginForm {
		t.Error("Expected login form to be detected")
	}
}

func TestAnalyzeURL_HTTPErrors(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{"OK", http.StatusOK, false},
		{"Not Found", http.StatusNotFound, true},
		{"Internal Server Error", http.StatusInternalServerError, true},
		{"Forbidden", http.StatusForbidden, true},
		{"Unauthorized", http.StatusUnauthorized, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					fmt.Fprint(w, "<html><head><title>OK</title></head></html>")
				}
			}))
			defer server.Close()

			analyzer := setupTestAnalyzer()
			result, err := analyzer.AnalyzeURL(context.Background(), server.URL)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for HTTP %d", tc.statusCode)
				}
				if result != nil {
					t.Error("Expected nil result for HTTP error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for HTTP %d: %v", tc.statusCode, err)
				}
				if result == nil {
					t.Error("Expected result for successful request")
				}
			}
		})
	}
}

func TestAnalyzeURL_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, "<html><title>Slow</title></html>")
	}))
	defer server.Close()

	// Create analyzer with short timeout
	cfg := config.AnalyzerConfig{
		RequestTimeout: 100 * time.Millisecond,
		LinkTimeout:    50 * time.Millisecond,
		MaxRedirects:   5,
		MaxWorkers:     3,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	analyzer := New(cfg, logger)

	result, err := analyzer.AnalyzeURL(context.Background(), server.URL)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if result != nil {
		t.Error("Expected nil result on timeout")
	}
}

func TestAnalyzeURL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(w, "<html><title>Slow</title></html>")
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result, err := analyzer.AnalyzeURL(ctx, server.URL)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if result != nil {
		t.Error("Expected nil result on context cancellation")
	}
}

func setupTestAnalyzer() *Analyzer {
	cfg := config.AnalyzerConfig{
		RequestTimeout: 5 * time.Second,
		LinkTimeout:    2 * time.Second,
		MaxRedirects:   5,
		MaxWorkers:     3,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return New(cfg, logger)
}

func TestDetectHTMLVersion(t *testing.T) {
	analyzer := setupTestAnalyzer()

	testCases := []struct {
		doctype  string
		expected string
	}{
		{"html", "HTML5"},
		{"HTML", "HTML5"},
		{"  html  ", "HTML5"}, // Test trimming
		{"html PUBLIC \"-//W3C//DTD HTML 4.01//EN\"", "HTML 4.01"},
		{"html PUBLIC \"-//W3C//DTD HTML 4.01 Transitional//EN\"", "HTML 4.01"},
		{"html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\"", "XHTML"},
		{"html PUBLIC \"-//W3C//DTD XHTML 1.1//EN\"", "XHTML"},
		{"unknown-doctype", "HTML5"}, // Default case
		{"", "HTML5"},                // Empty case
	}

	for _, tc := range testCases {
		t.Run(tc.doctype, func(t *testing.T) {
			result := analyzer.detectHTMLVersion(tc.doctype)
			if result != tc.expected {
				t.Errorf("detectHTMLVersion(%q) = %q, want %q", tc.doctype, result, tc.expected)
			}
		})
	}
}

func TestIsLoginForm_ValidLoginForms(t *testing.T) {
	analyzer := setupTestAnalyzer()

	validLoginForms := []struct {
		name string
		html string
	}{
		{
			name: "email and password",
			html: `<form>
				<input type="email" name="email">
				<input type="password" name="password">
			</form>`,
		},
		{
			name: "username and password",
			html: `<form>
				<input type="text" name="username">
				<input type="password" name="pass">
			</form>`,
		},
		{
			name: "login field and password",
			html: `<form>
				<input type="text" name="login">
				<input type="password" name="pwd">
			</form>`,
		},
		{
			name: "user field and password",
			html: `<form>
				<input type="text" name="user_name">
				<input type="password" name="password">
			</form>`,
		},
		{
			name: "implicit text type",
			html: `<form>
				<input name="email">
				<input type="password" name="password">
			</form>`,
		},
	}

	for _, tc := range validLoginForms {
		t.Run(tc.name, func(t *testing.T) {
			formNode := parseFormHTML(t, tc.html)

			result := analyzer.isLoginForm(formNode)
			if !result {
				t.Errorf("Expected login form to be detected for: %s", tc.name)
			}
		})
	}
}

func TestIsLoginForm_InvalidLoginForms(t *testing.T) {
	analyzer := setupTestAnalyzer()

	invalidLoginForms := []struct {
		name string
		html string
	}{
		{
			name: "search form",
			html: `<form>
				<input type="text" name="query">
				<input type="submit" value="Search">
			</form>`,
		},
		{
			name: "contact form",
			html: `<form>
				<input type="text" name="name">
				<input type="email" name="email">
				<textarea name="message"></textarea>
			</form>`,
		},
		{
			name: "form without password",
			html: `<form>
				<input type="text" name="username">
				<input type="text" name="message">
			</form>`,
		},
		{
			name: "form without username",
			html: `<form>
				<input type="password" name="password">
				<input type="text" name="other">
			</form>`,
		},
		{
			name: "empty form",
			html: `<form></form>`,
		},
	}

	for _, tc := range invalidLoginForms {
		t.Run(tc.name, func(t *testing.T) {
			formNode := parseFormHTML(t, tc.html)

			result := analyzer.isLoginForm(formNode)
			if result {
				t.Errorf("Expected login form NOT to be detected for: %s", tc.name)
			}
		})
	}
}

func TestProcessLink(t *testing.T) {
	analyzer := setupTestAnalyzer()
	baseURL, _ := url.Parse("https://example.com")

	testCases := []struct {
		name             string
		href             string
		expectedInternal int
		expectedExternal int
	}{
		{"relative path", "/about", 1, 0},
		{"relative with subdirectory", "/docs/api", 1, 0},
		{"absolute internal", "https://example.com/contact", 1, 0},
		{"external domain", "https://google.com", 0, 1},
		{"external subdomain", "https://api.example.com", 0, 1},
		{"external with path", "https://github.com/user/repo", 0, 1},
		{"query parameters", "/search?q=test", 1, 0},
		{"fragment", "/page#section", 1, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &Result{Headings: make(map[string]int)}

			linkNode := &html.Node{
				Type: html.ElementNode,
				Data: "a",
				Attr: []html.Attribute{{Key: "href", Val: tc.href}},
			}

			analyzer.processLink(linkNode, result, baseURL)

			if result.InternalLinks != tc.expectedInternal {
				t.Errorf("Expected %d internal links, got %d", tc.expectedInternal, result.InternalLinks)
			}

			if result.ExternalLinks != tc.expectedExternal {
				t.Errorf("Expected %d external links, got %d", tc.expectedExternal, result.ExternalLinks)
			}
		})
	}
}

func TestProcessLink_InvalidHref(t *testing.T) {
	analyzer := setupTestAnalyzer()
	baseURL, _ := url.Parse("https://example.com")
	result := &Result{Headings: make(map[string]int)}

	// Test invalid href
	linkNode := &html.Node{
		Type: html.ElementNode,
		Data: "a",
		Attr: []html.Attribute{{Key: "href", Val: "://invalid-url"}},
	}

	analyzer.processLink(linkNode, result, baseURL)

	// Should not increment either counter for invalid URLs
	if result.InternalLinks != 0 || result.ExternalLinks != 0 {
		t.Error("Expected no links to be counted for invalid href")
	}
}

func TestExtractLinks(t *testing.T) {
	analyzer := setupTestAnalyzer()
	baseURL, _ := url.Parse("https://example.com")

	testHTML := `<html>
		<body>
			<a href="/internal1">Internal 1</a>
			<a href="/internal2">Internal 2</a>
			<a href="https://external.com">External</a>
			<a href="mailto:test@example.com">Email</a>
			<a href="javascript:void(0)">JavaScript</a>
			<a href="ftp://files.example.com">FTP</a>
			<a href="https://example.com/page">Internal Absolute</a>
		</body>
	</html>`

	doc, err := html.Parse(strings.NewReader(testHTML))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	links := analyzer.extractLinks(doc, baseURL)

	// Should only extract HTTP/HTTPS links
	expectedCount := 4 // /internal1, /internal2, external.com, example.com/page
	if len(links) != expectedCount {
		t.Errorf("Expected %d links, got %d", expectedCount, len(links))
	}

	// Verify correct links are extracted
	expectedURLs := map[string]bool{
		"https://example.com/internal1": true,
		"https://example.com/internal2": true,
		"https://external.com":          true,
		"https://example.com/page":      true,
	}

	for _, link := range links {
		if !expectedURLs[link] {
			t.Errorf("Unexpected link extracted: %s", link)
		}
	}
}

func TestTraverseNode_ComplexHTML(t *testing.T) {
	analyzer := setupTestAnalyzer()
	baseURL, _ := url.Parse("https://example.com")

	complexHTML := `<!DOCTYPE html>
	<html>
	<head>
		<title>Complex Test Page</title>
	</head>
	<body>
		<h1>Main Title</h1>
		<div>
			<h2>Section 1</h2>
			<h2>Section 2</h2>
			<div>
				<h3>Subsection</h3>
				<h4>Sub-subsection 1</h4>
				<h4>Sub-subsection 2</h4>
				<h5>Deep section</h5>
			</div>
		</div>
		<nav>
			<a href="/page1">Page 1</a>
			<a href="/page2">Page 2</a>
			<a href="https://external.com">External</a>
		</nav>
		<form class="search">
			<input type="text" name="query">
		</form>
		<form class="login">
			<input type="email" name="email">
			<input type="password" name="password">
		</form>
	</body>
	</html>`

	doc, err := html.Parse(strings.NewReader(complexHTML))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	result := &Result{Headings: make(map[string]int)}
	analyzer.analyzeDocument(doc, result, baseURL)

	// Test title
	if result.Title != "Complex Test Page" {
		t.Errorf("Expected title 'Complex Test Page', got '%s'", result.Title)
	}

	// Test headings
	expectedHeadings := map[string]int{
		"h1": 1,
		"h2": 2,
		"h3": 1,
		"h4": 2,
		"h5": 1,
	}

	for level, expected := range expectedHeadings {
		if result.Headings[level] != expected {
			t.Errorf("Expected %d %s headings, got %d", expected, level, result.Headings[level])
		}
	}

	// Test links
	if result.InternalLinks != 2 {
		t.Errorf("Expected 2 internal links, got %d", result.InternalLinks)
	}

	if result.ExternalLinks != 1 {
		t.Errorf("Expected 1 external link, got %d", result.ExternalLinks)
	}

	// Test login form
	if !result.HasLoginForm {
		t.Error("Expected login form to be detected")
	}
}

// Helper function to parse form HTML and return form node
func parseFormHTML(t *testing.T, htmlString string) *html.Node {
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	var formNode *html.Node
	var findForm func(*html.Node)
	findForm = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			formNode = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findForm(c)
		}
	}
	findForm(doc)

	if formNode == nil {
		t.Fatal("Form node not found in HTML")
	}

	return formNode
}

func TestCheckSingleLink_StatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"OK", http.StatusOK, true},
		{"Created", http.StatusCreated, true},
		{"Found (redirect)", http.StatusFound, true},
		{"Moved Permanently", http.StatusMovedPermanently, true},
		{"Bad Request", http.StatusBadRequest, false},
		{"Not Found", http.StatusNotFound, false},
		{"Internal Server Error", http.StatusInternalServerError, false},
		{"Service Unavailable", http.StatusServiceUnavailable, false},
	}

	analyzer := setupTestAnalyzer()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify HEAD method is used
				if r.Method != http.MethodHead {
					t.Errorf("Expected HEAD request, got %s", r.Method)
				}

				// Verify User-Agent
				userAgent := r.Header.Get("User-Agent")
				if userAgent != "Web-Analyzer/1.0" {
					t.Errorf("Expected User-Agent 'Web-Analyzer/1.0', got '%s'", userAgent)
				}

				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := &http.Client{Timeout: 5 * time.Second}
			result := analyzer.checkSingleLink(context.Background(), client, server.URL)

			if result != tc.expected {
				t.Errorf("Expected %v for status %d, got %v", tc.expected, tc.statusCode, result)
			}
		})
	}
}

func TestCheckSingleLink_InvalidURL(t *testing.T) {
	analyzer := setupTestAnalyzer()
	client := &http.Client{Timeout: 5 * time.Second}

	invalidURLs := []string{
		"invalid-url",
		"://no-scheme",
		"http://",
		"",
	}

	for _, invalidURL := range invalidURLs {
		t.Run(fmt.Sprintf("invalid_%s", invalidURL), func(t *testing.T) {
			result := analyzer.checkSingleLink(context.Background(), client, invalidURL)

			if result {
				t.Errorf("Expected false for invalid URL: %s", invalidURL)
			}
		})
	}
}

func TestCheckSingleLink_NetworkError(t *testing.T) {
	analyzer := setupTestAnalyzer()
	client := &http.Client{Timeout: 5 * time.Second}

	// Use a non-existent domain
	result := analyzer.checkSingleLink(context.Background(), client, "http://definitely-does-not-exist-12345.com")

	if result {
		t.Error("Expected false for network error")
	}
}

func TestCheckLinksAccessibility_MixedResults(t *testing.T) {
	// Create accessible server
	accessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer accessibleServer.Close()

	// Create inaccessible server
	inaccessibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer inaccessibleServer.Close()

	analyzer := setupTestAnalyzer()
	links := []string{
		accessibleServer.URL,
		accessibleServer.URL + "/page1",
		inaccessibleServer.URL,
		"http://non-existent-domain-12345.com",
		accessibleServer.URL + "/page2",
	}

	inaccessibleCount := analyzer.checkLinksAccessibility(context.Background(), links)

	// Expect at least 2 inaccessible (404 server + invalid domain)
	if inaccessibleCount < 2 {
		t.Errorf("Expected at least 2 inaccessible links, got %d", inaccessibleCount)
	}

	// Should not exceed total links
	if inaccessibleCount > len(links) {
		t.Errorf("Inaccessible count (%d) cannot exceed total links (%d)", inaccessibleCount, len(links))
	}
}

func TestCheckLinksAccessibility_EmptyList(t *testing.T) {
	analyzer := setupTestAnalyzer()

	count := analyzer.checkLinksAccessibility(context.Background(), []string{})

	if count != 0 {
		t.Errorf("Expected 0 for empty links, got %d", count)
	}
}

func TestCheckLinksAccessibility_WorkerPoolLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()

	// Create fewer links than max workers to test worker limiting
	links := []string{server.URL, server.URL + "/page1"}

	count := analyzer.checkLinksAccessibility(context.Background(), links)

	// All should be accessible
	if count != 0 {
		t.Errorf("Expected 0 inaccessible links, got %d", count)
	}
}

func TestCheckLinksAccessibility_ContextCancellation(t *testing.T) {
	// Create a slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	analyzer := setupTestAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())

	links := []string{slowServer.URL, slowServer.URL + "/page1"}

	// Cancel context after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Should handle cancellation gracefully without panicking
	count := analyzer.checkLinksAccessibility(ctx, links)

	// The exact count may vary due to timing, but it shouldn't panic
	_ = count
}

func TestCheckLinksAccessibility_Concurrency(t *testing.T) {
	// Create multiple servers to test concurrent access
	servers := make([]*httptest.Server, 5)
	links := make([]string, 5)

	for i := 0; i < 5; i++ {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add small delay to simulate network latency
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		links[i] = servers[i].URL
	}

	defer func() {
		for _, server := range servers {
			server.Close()
		}
	}()

	analyzer := setupTestAnalyzer()
	start := time.Now()

	count := analyzer.checkLinksAccessibility(context.Background(), links)

	duration := time.Since(start)

	// All should be accessible
	if count != 0 {
		t.Errorf("Expected 0 inaccessible links, got %d", count)
	}

	// With concurrency, should complete faster than sequential (5 * 50ms = 250ms)
	// Allow some buffer for overhead
	if duration > 200*time.Millisecond {
		t.Errorf("Expected concurrent execution to be faster, took %v", duration)
	}
}
