package proxy

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/git-proxy/go/internal/config"
	"github.com/git-proxy/go/internal/logger"
	"github.com/git-proxy/go/internal/middleware"
)

type GitProxy struct {
	config      *config.Config
	logger      logger.Logger
	httpServer  *http.Server
	httpsServer *http.Server
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	started     bool
	mu          sync.RWMutex
}

func New(cfg *config.Config, log logger.Logger) *GitProxy {
	ctx, cancel := context.WithCancel(context.Background())
	return &GitProxy{
		config: cfg,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *GitProxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("server already started")
	}

	p.logger.Info("Starting Git Proxy server...")

	if p.config.Server.HTTPPort > 0 {
		if err := p.startHTTPServer(); err != nil {
			return fmt.Errorf("start HTTP server: %w", err)
		}
	}

	if p.config.Server.HTTPSPort > 0 {
		if err := p.startHTTPSServer(); err != nil {
			return fmt.Errorf("start HTTPS server: %w", err)
		}
	}

	p.started = true
	p.logger.Info("Git Proxy server started successfully")
	return nil
}

func (p *GitProxy) startHTTPServer() error {
	addr := fmt.Sprintf("%s:%d", p.config.Server.Host, p.config.Server.HTTPPort)

	proxy := p.createReverseProxy()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleGitRequest(proxy))
	mux.HandleFunc("/health", p.handleHealth)

	handler := middleware.Recovery(mux, p.logger)
	handler = middleware.RequestLogger(handler, p.logger)
	handler = middleware.CORS(handler)

	p.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
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

func (p *GitProxy) startHTTPSServer() error {
	addr := fmt.Sprintf("%s:%d", p.config.Server.Host, p.config.Server.HTTPSPort)

	if p.config.Server.CertFile == "" || p.config.Server.KeyFile == "" {
		p.logger.Warn("HTTPS enabled but no cert/key file provided, using HTTP")
		return p.startHTTPServer()
	}

	proxy := p.createReverseProxy()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleGitRequest(proxy))
	mux.HandleFunc("/health", p.handleHealth)

	handler := middleware.Recovery(mux, p.logger)
	handler = middleware.RequestLogger(handler, p.logger)
	handler = middleware.CORS(handler)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	p.httpsServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
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

func (p *GitProxy) createReverseProxy() *httputil.ReverseProxy {
	scheme := "https"
	if p.config.Server.HTTPSPort == 0 {
		scheme = "http"
	}

	targetURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", p.config.Target.Host, p.config.Target.Port),
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		if p.config.Target.Username != "" && p.config.Target.Password != "" {
			auth := base64.StdEncoding.EncodeToString(
				[]byte(p.config.Target.Username + ":" + p.config.Target.Password))
			req.Header.Set("Authorization", "Basic "+auth)
		}

		req.Header.Set("User-Agent", "git/2.40.0")

		p.logger.Debug("Proxying request: %s %s -> %s",
			req.Method, req.URL.Path, req.URL.String())
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("Proxy error: %v", err)
		http.Error(w, "Proxy error", http.StatusBadGateway)
	}

	return proxy
}

func (p *GitProxy) handleGitRequest(proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqPath := path.Clean(r.URL.Path)

		if !p.isPathAllowed(reqPath) {
			p.logger.Warn("Blocked path: %s", reqPath)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs"):
			p.handleInfoRefs(w, r, proxy)
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			p.handleGitUploadPack(w, r, proxy)
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			p.handleGitReceivePack(w, r, proxy)
		default:
			proxy.ServeHTTP(w, r)
		}
	}
}

func (p *GitProxy) isPathAllowed(reqPath string) bool {
	for _, blocked := range p.config.Proxy.BlockedPaths {
		if strings.HasPrefix(reqPath, path.Clean(blocked)) {
			return false
		}
	}

	if len(p.config.Proxy.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range p.config.Proxy.AllowedPaths {
			if strings.HasPrefix(reqPath, path.Clean(allowedPath)) {
				allowed = true
				break
			}
		}
		return allowed
	}

	return true
}

func (p *GitProxy) handleInfoRefs(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	service := r.URL.Query().Get("service")
	if service == "git-upload-pack" || service == "git-receive-pack" {
		p.logger.Info("Handling info-refs for service: %s", service)
	}
	proxy.ServeHTTP(w, r)
}

func (p *GitProxy) handleGitUploadPack(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	p.logger.Info("Handling git-upload-pack request: %s", r.URL.Path)
	proxy.ServeHTTP(w, r)
}

func (p *GitProxy) handleGitReceivePack(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	p.logger.Info("Handling git-receive-pack request: %s", r.URL.Path)
	proxy.ServeHTTP(w, r)
}

func (p *GitProxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (p *GitProxy) Stop() {
	p.logger.Info("Stopping Git Proxy server...")

	p.cancel()

	if p.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.httpServer.Shutdown(shutdownCtx)
	}

	if p.httpsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.httpsServer.Shutdown(shutdownCtx)
	}

	p.wg.Wait()
	p.logger.Info("Git Proxy server stopped")
}

func (p *GitProxy) WaitForSignal(sigChan <-chan os.Signal) {
	sig := <-sigChan
	p.logger.Info("Received signal: %s", sig)
	p.Stop()
}

func (p *GitProxy) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}

func (p *GitProxy) Context() context.Context {
	return p.ctx
}
