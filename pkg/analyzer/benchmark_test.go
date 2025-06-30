package analyzer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func BenchmarkAnalyzeURL_SimpleHTML(b *testing.B) {
	testHTML := `<!DOCTYPE html>
<html>
<head><title>Benchmark Test</title></head>
<body>
    <h1>Main Heading</h1>
    <h2>Section</h2>
    <a href="/page1">Link 1</a>
    <a href="https://external.com">External</a>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testHTML)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeURL(ctx, server.URL)
		if err != nil {
			b.Fatalf("AnalyzeURL failed: %v", err)
		}
	}
}

func BenchmarkAnalyzeURL_ComplexHTML(b *testing.B) {
	// Generate complex HTML with many elements
	htmlBuilder := `<!DOCTYPE html><html><head><title>Complex Benchmark</title></head><body>`

	// Add many headings
	for i := 1; i <= 10; i++ {
		htmlBuilder += fmt.Sprintf("<h1>Heading %d</h1>", i)
		htmlBuilder += fmt.Sprintf("<h2>Subheading %d.1</h2>", i)
		htmlBuilder += fmt.Sprintf("<h2>Subheading %d.2</h2>", i)
	}

	// Add many links
	for i := 1; i <= 50; i++ {
		htmlBuilder += fmt.Sprintf(`<a href="/page%d">Internal Link %d</a>`, i, i)
	}

	// Add external links
	for i := 1; i <= 10; i++ {
		htmlBuilder += fmt.Sprintf(`<a href="https://external%d.com">External %d</a>`, i, i)
	}

	htmlBuilder += `</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, htmlBuilder)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeURL(ctx, server.URL)
		if err != nil {
			b.Fatalf("AnalyzeURL failed: %v", err)
		}
	}
}

func BenchmarkAnalyzeURL_Parallel(b *testing.B) {
	testHTML := `<!DOCTYPE html>
<html>
<head><title>Parallel Test</title></head>
<body>
    <h1>Content</h1>
    <a href="/page">Link</a>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testHTML)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := analyzer.AnalyzeURL(ctx, server.URL)
			if err != nil {
				b.Fatalf("AnalyzeURL failed: %v", err)
			}
		}
	})
}

func BenchmarkCheckLinksAccessibility_SmallSet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()
	links := []string{
		server.URL + "/page1",
		server.URL + "/page2",
		server.URL + "/page3",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.checkLinksAccessibility(ctx, links)
	}
}

func BenchmarkCheckLinksAccessibility_LargeSet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	analyzer := setupTestAnalyzer()

	// Create 20 links to test
	links := make([]string, 20)
	for i := 0; i < 20; i++ {
		links[i] = fmt.Sprintf("%s/page%d", server.URL, i)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.checkLinksAccessibility(ctx, links)
	}
}

func BenchmarkDetectHTMLVersion(b *testing.B) {
	analyzer := setupTestAnalyzer()
	doctypes := []string{
		"html",
		"html PUBLIC \"-//W3C//DTD HTML 4.01//EN\"",
		"html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\"",
		"unknown-doctype",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, doctype := range doctypes {
			analyzer.detectHTMLVersion(doctype)
		}
	}
}

func BenchmarkExtractLinks(b *testing.B) {
	analyzer := setupTestAnalyzer()
	baseURL, _ := url.Parse("https://example.com")

	// Create HTML with many links
	htmlBuilder := "<html><body>"
	for i := 0; i < 100; i++ {
		htmlBuilder += fmt.Sprintf(`<a href="/page%d">Link %d</a>`, i, i)
	}
	htmlBuilder += "</body></html>"

	doc, err := html.Parse(strings.NewReader(htmlBuilder))
	if err != nil {
		b.Fatalf("Failed to parse HTML: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.extractLinks(doc, baseURL)
	}
}
