package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"citytree/internal/application"
)

//go:embed static/*
var assets embed.FS

type Server struct {
	service *application.Service
	mux     *http.ServeMux
	static  http.Handler
}

func NewServer(service *application.Service) *Server {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	s := &Server{service: service, mux: http.NewServeMux(), static: http.FileServer(http.FS(staticFS))}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return securityHeaders(s.mux) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.HandleIndex)
	s.mux.HandleFunc("GET /app.css", s.HandleStatic)
	s.mux.HandleFunc("GET /app.js", s.HandleStatic)
	s.mux.HandleFunc("GET /healthz", s.HandleHealth)
	s.mux.HandleFunc("GET /api/batches", s.HandleBatches)
	s.mux.HandleFunc("POST /api/batches", s.HandleBatches)
	s.mux.HandleFunc("GET /api/batches/{id}", s.HandleBatchDetail)
	s.mux.HandleFunc("POST /api/batches/{id}/trees", s.HandleAddTree)
	s.mux.HandleFunc("GET /api/trees/{id}", s.HandleTreeDetail)
	s.mux.HandleFunc("GET /api/trees/{id}/certificate", s.HandleCertificate)
	s.mux.HandleFunc("POST /api/trees/{id}/{action}", s.HandleTreeAction)
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := assets.ReadFile("static/index.html")
	_, _ = w.Write(data)
}

func (s *Server) HandleStatic(w http.ResponseWriter, r *http.Request) { s.static.ServeHTTP(w, r) }

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.service.Store().VerifyIntegrity(ctx); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
