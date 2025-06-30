package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"web-analyzer/internal/config"

	"golang.org/x/net/html"
)

// New creates a new analyzer instance
func New(config config.AnalyzerConfig, logger *slog.Logger) *Analyzer {
	return &Analyzer{
		client: &http.Client{
			Timeout: config.RequestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= config.MaxRedirects {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		config: config,
		logger: logger,
	}
}

// AnalyzeURL analyzes a web page and returns results
func (a *Analyzer) AnalyzeURL(ctx context.Context, targetURL string) (*Result, error) {
	start := time.Now()

	a.logger.Debug("Starting URL analysis", "url", targetURL)

	result := &Result{
		URL:      targetURL,
		Headings: make(map[string]int),
	}

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		a.logger.Error("URL parsing failed", "url", targetURL, "error", err)
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		targetURL = "http://" + targetURL
		parsedURL, err = url.Parse(targetURL)
		if err != nil {
			a.logger.Error("URL normalization failed", "url", targetURL, "error", err)
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		a.logger.Debug("URL normalized", "original", result.URL, "normalized", targetURL)
	}

	result.URL = targetURL

	// Fetch HTML content
	doc, err := a.fetchHTML(ctx, targetURL)
	if err != nil {
		a.logger.Error("HTML fetch failed", "url", targetURL, "error", err)
		return nil, fmt.Errorf("failed to fetch HTML: %w", err)
	}

	a.logger.Debug("HTML fetched successfully", "url", targetURL)

	// Analyze document
	a.analyzeDocument(doc, result, parsedURL)

	// Check link accessibility
	links := a.extractLinks(doc, parsedURL)
	linkCount := len(links)

	if linkCount > 0 {
		a.logger.Debug("Starting link accessibility check",
			"url", targetURL,
			"total_links", linkCount,
			"max_workers", a.config.MaxWorkers,
		)

		result.InaccessibleLinks = a.checkLinksAccessibility(ctx, links)

		a.logger.Debug("Link accessibility check completed",
			"url", targetURL,
			"total_links", linkCount,
			"inaccessible", result.InaccessibleLinks,
		)
	}

	duration := time.Since(start)

	a.logger.Info("URL analysis completed",
		"url", targetURL,
		"duration", duration,
		"html_version", result.HTMLVersion,
		"title", result.Title,
		"headings", result.Headings,
		"internal_links", result.InternalLinks,
		"external_links", result.ExternalLinks,
		"inaccessible_links", result.InaccessibleLinks,
		"has_login_form", result.HasLoginForm,
	)

	return result, nil
}

// fetchHTML fetches and parses HTML from URL
func (a *Analyzer) fetchHTML(ctx context.Context, targetURL string) (*html.Node, error) {
	a.logger.Debug("Creating HTTP request", "url", targetURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Web-Analyzer/1.0")

	a.logger.Debug("Sending HTTP request", "url", targetURL)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	a.logger.Debug("Received HTTP response",
		"url", targetURL,
		"status", resp.StatusCode,
		"content_type", resp.Header.Get("Content-Type"),
		"content_length", resp.Header.Get("Content-Length"),
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	return doc, nil
}

// analyzeDocument analyzes the HTML document
func (a *Analyzer) analyzeDocument(doc *html.Node, result *Result, baseURL *url.URL) {
	a.logger.Debug("Starting document analysis", "url", baseURL.String())
	a.traverseNode(doc, result, baseURL)
	a.logger.Debug("Document analysis completed",
		"url", baseURL.String(),
		"title", result.Title,
		"headings", result.Headings,
	)
}

// traverseNode recursively traverses HTML nodes
func (a *Analyzer) traverseNode(n *html.Node, result *Result, baseURL *url.URL) {
	if n.Type == html.ElementNode {
		switch strings.ToLower(n.Data) {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				result.Title = strings.TrimSpace(n.FirstChild.Data)
				a.logger.Debug("Found page title", "title", result.Title)
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := strings.ToLower(n.Data)
			result.Headings[level]++
			a.logger.Debug("Found heading", "level", level, "count", result.Headings[level])
		case "a":
			a.processLink(n, result, baseURL)
		case "form":
			if a.isLoginForm(n) {
				result.HasLoginForm = true
				a.logger.Debug("Login form detected")
			}
		}
	} else if n.Type == html.DoctypeNode {
		result.HTMLVersion = a.detectHTMLVersion(n.Data)
		a.logger.Debug("HTML version detected", "version", result.HTMLVersion)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		a.traverseNode(c, result, baseURL)
	}
}

// processLink processes anchor tags
func (a *Analyzer) processLink(n *html.Node, result *Result, baseURL *url.URL) {
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			linkURL, err := url.Parse(attr.Val)
			if err != nil {
				a.logger.Debug("Invalid link URL", "href", attr.Val, "error", err)
				continue
			}

			resolvedURL := baseURL.ResolveReference(linkURL)

			if resolvedURL.Host == baseURL.Host {
				result.InternalLinks++
				a.logger.Debug("Internal link found", "href", resolvedURL.String())
			} else {
				result.ExternalLinks++
				a.logger.Debug("External link found", "href", resolvedURL.String())
			}
			break
		}
	}
}

// isLoginForm determines if a form is a login form
func (a *Analyzer) isLoginForm(n *html.Node) bool {
	hasPasswordField := false
	hasUsernameField := false

	a.checkFormFields(n, &hasPasswordField, &hasUsernameField)

	isLogin := hasPasswordField && hasUsernameField
	a.logger.Debug("Form analysis",
		"has_password", hasPasswordField,
		"has_username", hasUsernameField,
		"is_login_form", isLogin,
	)

	return isLogin
}

// checkFormFields recursively checks form fields
func (a *Analyzer) checkFormFields(n *html.Node, hasPassword, hasUsername *bool) {
	if n.Type == html.ElementNode && n.Data == "input" {
		inputType := ""
		inputName := ""

		for _, attr := range n.Attr {
			if attr.Key == "type" {
				inputType = strings.ToLower(attr.Val)
			}
			if attr.Key == "name" {
				inputName = strings.ToLower(attr.Val)
			}
		}

		if inputType == "password" {
			*hasPassword = true
			a.logger.Debug("Password field found", "name", inputName)
		}

		if inputType == "text" || inputType == "email" || inputType == "" {
			if strings.Contains(inputName, "user") || strings.Contains(inputName, "email") ||
				strings.Contains(inputName, "login") {
				*hasUsername = true
				a.logger.Debug("Username field found", "name", inputName, "type", inputType)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		a.checkFormFields(c, hasPassword, hasUsername)
	}
}

// detectHTMLVersion determines HTML version from DOCTYPE
func (a *Analyzer) detectHTMLVersion(doctype string) string {
	doctype = strings.ToLower(strings.TrimSpace(doctype))

	if doctype == "html" {
		return "HTML5"
	}

	if strings.Contains(doctype, "xhtml") {
		return "XHTML"
	}

	if strings.Contains(doctype, "html 4") {
		return "HTML 4.01"
	}

	return "HTML5" // Default
}

// extractLinks extracts all links from the document
func (a *Analyzer) extractLinks(doc *html.Node, baseURL *url.URL) []string {
	var links []string
	a.extractLinksFromNode(doc, baseURL, &links)
	a.logger.Debug("Links extracted", "count", len(links))
	return links
}

// extractLinksFromNode recursively extracts links
func (a *Analyzer) extractLinksFromNode(n *html.Node, baseURL *url.URL, links *[]string) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				linkURL, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}

				resolvedURL := baseURL.ResolveReference(linkURL)
				if resolvedURL.Scheme == "http" || resolvedURL.Scheme == "https" {
					*links = append(*links, resolvedURL.String())
				}
				break
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		a.extractLinksFromNode(c, baseURL, links)
	}
}

// checkLinksAccessibility checks accessibility of links with configurable concurrency
func (a *Analyzer) checkLinksAccessibility(ctx context.Context, links []string) int {
	if len(links) == 0 {
		return 0
	}

	maxWorkers := a.config.MaxWorkers
	if maxWorkers > len(links) {
		maxWorkers = len(links)
	}

	a.logger.Debug("Starting concurrent link checking",
		"total_links", len(links),
		"workers", maxWorkers,
		"timeout", a.config.LinkTimeout,
	)

	client := &http.Client{
		Timeout: a.config.LinkTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= a.config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	jobs := make(chan string, len(links))
	results := make(chan bool, len(links))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			a.logger.Debug("Link checker worker started", "worker_id", workerID)

			linksChecked := 0
			for url := range jobs {
				accessible := a.checkSingleLink(ctx, client, url)
				results <- accessible
				linksChecked++

				a.logger.Debug("Link checked",
					"worker_id", workerID,
					"url", url,
					"accessible", accessible,
					"checked_count", linksChecked,
				)
			}

			a.logger.Debug("Link checker worker finished",
				"worker_id", workerID,
				"links_checked", linksChecked,
			)
		}(i)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, link := range links {
			select {
			case jobs <- link:
			case <-ctx.Done():
				a.logger.Warn("Context cancelled while sending jobs")
				return
			}
		}
	}()

	// Close results when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	inaccessible := 0
	processed := 0
	for accessible := range results {
		processed++
		if !accessible {
			inaccessible++
		}
	}

	a.logger.Info("Link accessibility check completed",
		"total_links", len(links),
		"processed", processed,
		"accessible", processed-inaccessible,
		"inaccessible", inaccessible,
		"workers_used", maxWorkers,
	)

	return inaccessible
}

// checkSingleLink checks if a single link is accessible
func (a *Analyzer) checkSingleLink(ctx context.Context, client *http.Client, link string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		a.logger.Debug("Failed to create request for link", "url", link, "error", err)
		return false
	}

	req.Header.Set("User-Agent", "Web-Analyzer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		a.logger.Debug("Link check failed", "url", link, "error", err)
		return false
	}
	defer resp.Body.Close()

	accessible := resp.StatusCode >= 200 && resp.StatusCode < 400

	a.logger.Debug("Link checked",
		"url", link,
		"status", resp.StatusCode,
		"accessible", accessible,
	)

	return accessible
}
