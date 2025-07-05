# Web Page Analyzer

A Go-based web application that analyzes web pages and provides detailed information about their HTML structure, links, and forms following the test task requirements.

## Project Overview

This web application allows users to analyze any publicly accessible web page by entering a URL. The analyzer extracts key information including HTML version, page structure, link analysis, and form detection. Built with clean Go architecture, comprehensive testing, and following Go best practices.

## Features

- **HTML Version Detection**: Automatically detects HTML5, HTML 4.01, and XHTML
- **Page Structure Analysis**: Extracts page title and counts headings by level (h1-h6)
- **Link Analysis**: Identifies and counts internal/external links with accessibility checking
- **Form Detection**: Automatically detects login forms based on field patterns
- **Error Handling**: Comprehensive error reporting with HTTP status codes
- **Concurrent Processing**: Multi-worker link accessibility checking for performance
- **Real-time Analysis**: Fast response times with proper timeout handling

## Technology Stack

### Backend
- **Language**: Go 1.24+
- **Web Framework**: Standard `net/http` package
- **HTML Parsing**: `golang.org/x/net/html`
- **Logging**: `log/slog` (structured logging)
- **Testing**: Go standard testing package with `httptest`

### Frontend
- **Template Engine**: Go `html/template`
- **Styling**: Minimal CSS for clean, responsive UI
- **No JavaScript**: Pure server-side rendering as per requirements

### DevOps
- **Containerization**: Docker
- **Build Tool**: Makefile
- **Dependency Management**: Go modules

## Prerequisites

- Go 1.24 or higher
- Docker (optional, for containerized deployment)
- Git


## Installation & Setup

### Local Development

1. **Clone the repository**
```bash
git clone https://github.com/anjula-paulus/web-analyzer.git
cd web-analyzer
```

2. **Install dependencies**
```bash
go mod download
```

3. **Run the application**
```bash
# Development run
make run

# Or with custom configuration
make run_dev

# Or with environment variables
make run_env
```

4. **Access the application**
```
http://localhost:8080
```

### Using Makefile

```bash
# Build the application
make build

# Build for different platforms
make build_windows    # Windows executable
make build_linux      # Linux executable  
make build_darwin     # macOS executable

# Build and run immediately
make build_and_run

# Run locally (development)
make run

# Run with custom config file
make run_dev

# Run with environment variables
make run_env

# Run tests
make test

# Clean build artifacts
make clean
```

### Docker Deployment

1. **Build and run with Docker**
```bash
# Build and run (port 8080)
make docker_run

# Build image only
make docker_build

# Run with custom port
make docker_run_custom
```

2. **Manual Docker commands**
```bash
# Build Docker image
docker build -t web-analyzer .

# Run container (default port 8080)
docker run -p 8080:8080 web-analyzer

# Run with custom port and environment
docker run -p 9090:9090 -e PORT=:9090 web-analyzer
```

**Exposed Ports:**
- `8080` - Main web application
- `6060` - pprof debugging endpoint (if enabled)

## Configuration

The application supports multiple configuration methods:

### Configuration File (`config.yaml`)
```yaml
server:
  port: ":8080"
  read_timeout: "30s"
  write_timeout: "30s"

analyzer:
  request_timeout: "30s"
  link_timeout: "10s"
  max_redirects: 5
  max_workers: 10

logging:
  level: "info"
  format: "json"
```

### Runtime Configuration Options
```bash
# Use custom config file
CONFIG_PATH=config.yaml go run ./cmd/web-analyzer

# Override with environment variables
PORT=:9090 MAX_WORKERS=20 go run ./cmd/web-analyzer
```

### Configuration Structure

```go
type AnalyzerConfig struct {
    RequestTimeout time.Duration // HTTP request timeout
    LinkTimeout    time.Duration // Individual link check timeout  
    MaxRedirects   int          // Maximum redirects to follow
    MaxWorkers     int          // Concurrent workers for link checking
}
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Main analysis form |
| `/analyze` | POST | Submit URL for analysis |
| `/health` | GET | Health check endpoint |

## Usage

### Main Functionality

1. **Web Page Analysis**
   - Enter any valid URL in the form
   - Click "Analyze" to process the page
   - View comprehensive analysis results

2. **Analysis Results Include**
   - HTML version detection
   - Page title extraction
   - Heading count by level (h1, h2, h3, etc.)
   - Internal vs external link classification
   - Link accessibility status
   - Login form detection

3. **Error Handling**
   - Invalid URL format validation
   - Network timeout handling
   - HTTP error status reporting with codes
   - User-friendly error messages

### Example Analysis Output

```json
{
  "url": "https://example.com",
  "html_version": "HTML5",
  "title": "Example Domain",
  "headings": {
    "h1": 1,
    "h2": 0,
    "h3": 0
  },
  "internal_links": 0,
  "external_links": 1,
  "inaccessible_links": 0,
  "has_login_form": false
}
```
## Key Implementation Decisions

### Architecture Choices

1. **Clean Architecture**: Separated concerns with distinct packages
2. **Standard Library Focus**: Minimal external dependencies (`golang.org/x/net/html` only)
3. **Concurrent Design**: Worker pools with channels for link checking
4. **Context-First**: All operations support context cancellation

### Error Handling Strategy

- **Structured Logging**: Using `slog` for consistent log format
- **Wrapped Errors**: Proper error context with `fmt.Errorf`
- **User-Friendly Messages**: Clear descriptions with HTTP status codes
- **Graceful Degradation**: Analysis continues even if some operations fail

### Go Best Practices Applied

- **Idiomatic Go**: No Java-style patterns
- **Interface Segregation**: Small, focused interfaces
- **Error Handling**: Explicit error checking and proper propagation
- **Concurrency**: Proper use of channels and wait groups
- **Testing**: Table-driven tests and comprehensive coverage

## Challenges & Solutions

### 1. Concurrent Link Checking
**Challenge**: Checking hundreds of links sequentially is slow  
**Solution**: Implemented worker pool pattern with configurable concurrency using channels and goroutines

### 2. Memory Management
**Challenge**: Large pages could consume excessive memory  
**Solution**: Streaming HTML parsing with `golang.org/x/net/html` and bounded worker queues

### 3. Timeout Handling
**Challenge**: Preventing hanging requests on slow/unresponsive sites  
**Solution**: Context-based timeouts at multiple levels (request, link checking) with proper cancellation

### 4. Form Detection Accuracy
**Challenge**: Reliably identifying login forms vs other forms  
**Solution**: Pattern matching on input types and field names (password + username/email)

### 5. URL Validation
**Challenge**: Handling various URL formats and schemes  
**Solution**: Go's standard `net/url` package with automatic scheme normalization

## Possible Improvements

### Performance Enhancements
- **Result Caching**: Add Redis for analyzed page results
- **Rate Limiting**: Implement request limiting per IP using `golang.org/x/time/rate`
- **Connection Pooling**: Optimize HTTP client connection reuse
- **Batch Processing**: Support analyzing multiple URLs concurrently