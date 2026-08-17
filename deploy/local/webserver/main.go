package main

import (
	"context"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"regexp"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	listenAddress = ":8080"
	documentRoot  = "/srv"
)

var contentHashedAsset = regexp.MustCompile(`^/assets/[A-Za-z0-9_.-]+-[A-Za-z0-9_-]{6,}\.(?:css|js|map|svg|png|jpg|jpeg|webp|ttf|woff|woff2)$`)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	server := &http.Server{
		Addr:              listenAddress,
		Handler:           routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	serverError := make(chan error, 1)
	go func() {
		serverError <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("web server shutdown: %v", err)
			os.Exit(1)
		}
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("web server: %v", err)
			os.Exit(1)
		}
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", health)
	mux.HandleFunc("/health/ready", health)
	mux.Handle("/", http.HandlerFunc(serveArtifact))
	return mux
}

func health(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = io.WriteString(response, "ok\n")
	}
}

func serveArtifact(response http.ResponseWriter, request *http.Request) {
	setArtifactCachePolicy(response, request.URL.Path)
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relativePath := strings.TrimPrefix(filepath.Clean("/"+request.URL.Path), "/")
	candidate := filepath.Join(documentRoot, relativePath)
	if relativePath == "" {
		candidate = filepath.Join(documentRoot, "index.html")
	}
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		serveFile(response, request, candidate)
		return
	}
	if acceptsHTML(request) && !strings.HasPrefix(relativePath, "assets/") {
		serveFile(response, request, filepath.Join(documentRoot, "index.html"))
		return
	}
	http.NotFound(response, request)
}

func serveFile(response http.ResponseWriter, request *http.Request, filename string) {
	if filepath.Ext(filename) == ".html" {
		response.Header().Set("Cache-Control", "no-store, no-transform")
	}
	if contentType := mime.TypeByExtension(filepath.Ext(filename)); contentType != "" {
		response.Header().Set("Content-Type", contentType)
	}

	// http.ServeFile applies its own directory/index redirect policy. In
	// particular, a request for /index.html is redirected to ./ before the
	// body is served. That redirect is harmless for a normal document request
	// but WebKit rejects it when the response is returned by the app-shell
	// service worker during an OAuth callback navigation. Serve the opened file
	// directly so /index.html remains a 200 response with no Location header.
	file, err := os.Open(filename)
	if err != nil {
		http.NotFound(response, request)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(response, request)
		return
	}
	http.ServeContent(response, request, info.Name(), info.ModTime(), file)
}

func setArtifactCachePolicy(response http.ResponseWriter, pathname string) {
	switch {
	case pathname == "/" || pathname == "/index.html" || pathname == "/sw.js" || pathname == "/app-shell-assets.json" || pathname == "/http-config.json" || pathname == "/demo-build.json":
		response.Header().Set("Cache-Control", "no-store, no-transform")
	case contentHashedAsset.MatchString(pathname):
		response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case strings.HasPrefix(pathname, "/assets/") || strings.HasPrefix(pathname, "/api/") || strings.HasPrefix(pathname, "/v1/") || strings.HasPrefix(pathname, "/auth/") || strings.HasPrefix(pathname, "/identity/") || strings.HasPrefix(pathname, "/health/") || strings.HasPrefix(pathname, "/operations/") || strings.HasPrefix(pathname, "/otel/") || strings.HasPrefix(pathname, "/private/") || strings.HasPrefix(pathname, "/evidence-") || strings.HasPrefix(pathname, "/inspection-attachments/") || strings.HasPrefix(pathname, "/generated-documents/"):
		response.Header().Set("Cache-Control", "no-store, no-transform")
	}
}

func acceptsHTML(request *http.Request) bool {
	return request.Header.Get("Accept") == "" ||
		strings.Contains(request.Header.Get("Accept"), "text/html")
}

func runHealthcheck() {
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:8080/health/ready")
	if err != nil {
		os.Exit(1)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
