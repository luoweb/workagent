package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the git proxy
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Target  TargetConfig  `yaml:"target"`
	Logging LoggingConfig `yaml:"logging"`
	Proxy   ProxyConfig   `yaml:"proxy"`
}

// ServerConfig configures the proxy server
type ServerConfig struct {
	Host      string `yaml:"host"`
	HTTPPort  int    `yaml:"http_port"`
	HTTPSPort int    `yaml:"https_port"`
	CertFile  string `yaml:"cert_file"`
	KeyFile   string `yaml:"key_file"`
}

// TargetConfig configures the upstream Git server
type TargetConfig struct {
	Host     string       `yaml:"host"`
	Port     int          `yaml:"port"`
	Username string       `yaml:"username"`
	Password string       `yaml:"password"`
	SSH      SSHConfig    `yaml:"ssh"`
}

// SSHConfig configures SSH proxy behavior
type SSHConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ProxyConfig configures proxy behavior
type ProxyConfig struct {
	Timeout        int      `yaml:"timeout"`
	MaxConnections int      `yaml:"max_connections"`
	PathRewrite    bool     `yaml:"path_rewrite"`
	AllowedPaths   []string `yaml:"allowed_paths"`
	BlockedPaths   []string `yaml:"blocked_paths"`
}

// Logger is a simple logger interface
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

// SimpleLogger implements Logger with level control
type SimpleLogger struct {
	level  int
	format string
}

const (
	DEBUG = 0
	INFO  = 1
	WARN  = 2
	ERROR = 3
)

func parseLevel(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

func (l *SimpleLogger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func (l *SimpleLogger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

func (l *SimpleLogger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		log.Printf("[WARN] "+format, v...)
	}
}

func (l *SimpleLogger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

// GitProxy represents the main proxy server
type GitProxy struct {
	config     *Config
	logger     Logger
	httpServer *http.Server
	httpsServer *http.Server
	sshListener net.Listener
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewGitProxy creates a new GitProxy instance
func NewGitProxy(configPath string) (*GitProxy, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	level := parseLevel(config.Logging.Level)
	logger := &SimpleLogger{level: level, format: config.Logging.Format}

	ctx, cancel := context.WithCancel(context.Background())

	return &GitProxy{
		config:  config,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// loadConfig loads configuration from YAML file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if config.Proxy.Timeout <= 0 {
		config.Proxy.Timeout = 30
	}
	if config.Proxy.MaxConnections <= 0 {
		config.Proxy.MaxConnections = 100
	}

	return &config, nil
}

// Start starts the git proxy server
func (p *GitProxy) Start() error {
	p.logger.Info("Starting Git Proxy server...")

	// Start HTTP server
	if p.config.Server.HTTPPort > 0 {
		if err := p.startHTTPServer(); err != nil {
			return fmt.Errorf("start HTTP server: %w", err)
		}
	}

	// Start HTTPS server
	if p.config.Server.HTTPSPort > 0 {
		if err := p.startHTTPSServer(); err != nil {
			return fmt.Errorf("start HTTPS server: %w", err)
		}
	}

	// Start SSH proxy if enabled
	if p.config.Target.SSH.Enabled {
		if err := p.startSSHProxy(); err != nil {
			return fmt.Errorf("start SSH proxy: %w", err)
		}
	}

	p.logger.Info("Git Proxy server started successfully")
	return nil
}

// startHTTPServer starts the HTTP proxy server
func (p *GitProxy) startHTTPServer() error {
	addr := fmt.Sprintf("%s:%d", p.config.Server.Host, p.config.Server.HTTPPort)
	
	proxy := p.createReverseProxy()
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleGitRequest(proxy))
	
	p.httpServer = &http.Server{
		Addr:         addr,
		Handler:      p.loggingMiddleware(mux),
		ReadTimeout:  time.Duration(p.config.Proxy.Timeout) * time.Second,
		WriteTimeout: time.Duration(p.config.Proxy.Timeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.logger.Info("HTTP server listening on %s", addr)
		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error("HTTP server error: %v", err)
		}
	}()

	return nil
}

// startHTTPSServer starts the HTTPS proxy server
func (p *GitProxy) startHTTPSServer() error {
	addr := fmt.Sprintf("%s:%d", p.config.Server.Host, p.config.Server.HTTPSPort)

	if p.config.Server.CertFile == "" || p.config.Server.KeyFile == "" {
		p.logger.Warn("HTTPS enabled but no cert/key file provided, using HTTP")
		return p.startHTTPServer()
	}

	proxy := p.createReverseProxy()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleGitRequest(proxy))

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	p.httpsServer = &http.Server{
		Addr:         addr,
		Handler:      p.loggingMiddleware(mux),
		ReadTimeout:  time.Duration(p.config.Proxy.Timeout) * time.Second,
		WriteTimeout: time.Duration(p.config.Proxy.Timeout) * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig:    tlsConfig,
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.logger.Info("HTTPS server listening on %s", addr)
		if err := p.httpsServer.ListenAndServeTLS(p.config.Server.CertFile, p.config.Server.KeyFile); err != nil && err != http.ErrServerClosed {
			p.logger.Error("HTTPS server error: %v", err)
		}
	}()

	return nil
}

// createReverseProxy creates a reverse proxy for HTTP/HTTPS traffic
func (p *GitProxy) createReverseProxy() *httputil.ReverseProxy {
	targetURL := &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%d", p.config.Target.Host, p.config.Target.Port),
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Custom director to modify requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Add authentication if configured
		if p.config.Target.Username != "" && p.config.Target.Password != "" {
			auth := base64.StdEncoding.EncodeToString(
				[]byte(p.config.Target.Username + ":" + p.config.Target.Password))
			req.Header.Set("Authorization", "Basic "+auth)
		}

		// Set User-Agent to mimic git client
		req.Header.Set("User-Agent", "git/2.40.0")

		p.logger.Debug("Proxying request: %s %s -> %s",
			req.Method, req.URL.Path, req.URL.String())
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("Proxy error: %v", err)
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
	}

	return proxy
}

// handleGitRequest handles incoming git requests
func (p *GitProxy) handleGitRequest(proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if path is allowed
		if !p.isPathAllowed(r.URL.Path) {
			p.logger.Warn("Blocked path: %s", r.URL.Path)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Handle git protocol requests
		switch r.URL.Path {
		case "/info/refs":
			p.handleInfoRefs(w, r, proxy)
		case "/git-upload-pack":
			p.handleGitUploadPack(w, r, proxy)
		case "/git-receive-pack":
			p.handleGitReceivePack(w, r, proxy)
		default:
			// Check for path-based git operations
			if strings.Contains(r.URL.Path, "/git-upload-pack") ||
				strings.Contains(r.URL.Path, "/git-receive-pack") {
				proxy.ServeHTTP(w, r)
				return
			}
			// Default: use reverse proxy
			proxy.ServeHTTP(w, r)
		}
	}
}

// isPathAllowed checks if the request path is allowed
func (p *GitProxy) isPathAllowed(reqPath string) bool {
	// Check blocked paths
	for _, blocked := range p.config.Proxy.BlockedPaths {
		if strings.HasPrefix(reqPath, blocked) {
			return false
		}
	}

	// Check allowed paths
	if len(p.config.Proxy.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range p.config.Proxy.AllowedPaths {
			if strings.HasPrefix(reqPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

// handleInfoRefs handles git-info-refs service
func (p *GitProxy) handleInfoRefs(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	service := r.URL.Query().Get("service")
	if service == "git-upload-pack" || service == "git-receive-pack" {
		p.logger.Info("Handling info-refs for service: %s", service)
	}
	proxy.ServeHTTP(w, r)
}

// handleGitUploadPack handles git-upload-pack (fetch/clone)
func (p *GitProxy) handleGitUploadPack(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	p.logger.Info("Handling git-upload-pack request: %s", r.URL.Path)
	proxy.ServeHTTP(w, r)
}

// handleGitReceivePack handles git-receive-pack (push)
func (p *GitProxy) handleGitReceivePack(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	p.logger.Info("Handling git-receive-pack request: %s", r.URL.Path)
	proxy.ServeHTTP(w, r)
}

// startSSHProxy starts the SSH proxy server
func (p *GitProxy) startSSHProxy() error {
	// SSH proxy typically runs on a separate port
	// For simplicity, we'll use the same port as configured
	// In production, this would be port 22 or a dedicated port
	
	addr := fmt.Sprintf("%s:%d", p.config.Server.Host, p.config.Target.SSH.Port)
	
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	p.sshListener = ln

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.logger.Info("SSH proxy listening on %s", addr)
		p.handleSSHConnections()
	}()

	return nil
}

// handleSSHConnections handles incoming SSH connections
func (p *GitProxy) handleSSHConnections() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		p.sshListener.SetDeadline(time.Now().Add(10 * time.Second))
		
		conn, err := p.sshListener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			if err != io.EOF {
				p.logger.Error("SSH accept error: %v", err)
			}
			return
		}

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.handleSSHConnection(conn)
		}()
	}
}

// handleSSHConnection handles a single SSH connection
func (p *GitProxy) handleSSHConnection(conn net.Conn) {
	defer conn.Close()

	p.logger.Debug("New SSH connection from %s", conn.RemoteAddr())

	// Read SSH version string
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		p.logger.Error("Failed to read SSH version: %v", err)
		return
	}

	p.logger.Debug("SSH version: %s", strings.TrimSpace(line))

	// Connect to target SSH server
	targetAddr := fmt.Sprintf("%s:%d", p.config.Target.Host, p.config.Target.SSH.Port)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		p.logger.Error("Failed to connect to target SSH: %v", err)
		return
	}
	defer targetConn.Close()

	// Write our version string to target
	targetConn.Write([]byte(line))

	// Bidirectional copy of data
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(targetConn, conn)
		targetConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, targetConn)
		conn.Close()
	}()

	wg.Wait()
	p.logger.Debug("SSH connection closed")
}

// loggingMiddleware logs HTTP requests
func (p *GitProxy) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		p.logger.Info("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Stop stops the git proxy server
func (p *GitProxy) Stop() {
	p.logger.Info("Stopping Git Proxy server...")
	
	p.cancel()

	// Shutdown HTTP server
	if p.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.httpServer.Shutdown(shutdownCtx)
	}

	// Shutdown HTTPS server
	if p.httpsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.httpsServer.Shutdown(shutdownCtx)
	}

	// Close SSH listener
	if p.sshListener != nil {
		p.sshListener.Close()
	}

	p.wg.Wait()
	p.logger.Info("Git Proxy server stopped")
}

// WaitForSignal waits for termination signals
func (p *GitProxy) WaitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	sig := <-sigChan
	p.logger.Info("Received signal: %s", sig)
	p.Stop()
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Change to config file directory for relative paths
	if *configPath != "" {
		dir := path.Dir(*configPath)
		if dir != "." {
			os.Chdir(dir)
		}
		*configPath = path.Base(*configPath)
	}

	proxy, err := NewGitProxy(*configPath)
	if err != nil {
		log.Fatalf("Failed to create git proxy: %v", err)
	}

	if err := proxy.Start(); err != nil {
		log.Fatalf("Failed to start git proxy: %v", err)
	}

	proxy.WaitForSignal()
}
