package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const allowedOrigin = "http://localhost:3000"

type Server struct {
	store   Store
	cache   *LRU
	limiter *RateLimiter
	mux     *http.ServeMux
}

func NewServer(store Store) *Server {
	s := &Server{store: store, cache: NewLRU(100_000), limiter: NewRateLimiter(10, time.Minute), mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /links", s.createLink)
	s.mux.HandleFunc("OPTIONS /links", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	s.mux.HandleFunc("GET /{code}", s.redirect)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") == allowedOrigin {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", allowedOrigin)
		h.Set("Vary", "Origin")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		h.Set("Access-Control-Max-Age", "600")
	}
	s.mux.ServeHTTP(w, r)
}

type createReq struct {
	URL  string `json:"url"`
	Code string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.Allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many links created; try again in a minute.")
		return
	}
	var req createReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_URL", "Request body must be JSON with a url and code.")
		return
	}
	u, ok := validURL(req.URL)
	if !ok {
		writeErr(w, http.StatusBadRequest, "INVALID_URL", "URL must be a well-formed http or https URL of at most 2048 characters.")
		return
	}
	if isSelfReference(u) {
		writeErr(w, http.StatusBadRequest, "SELF_REFERENCE", "URL cannot point to this shortener.")
		return
	}
	code := strings.ToLower(req.Code)
	if !validCode(code) {
		writeErr(w, http.StatusBadRequest, "INVALID_CODE", "Code must be 6-10 characters: letters, digits, - or _.")
		return
	}
	if isReserved(code) {
		writeErr(w, http.StatusBadRequest, "RESERVED_CODE", "This code is reserved; choose another.")
		return
	}
	created, stored, err := s.store.Create(r.Context(), code, req.URL)
	if err != nil {
		log.Printf("create %q: %v", code, err)
		writeErr(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Service temporarily unavailable; please retry.")
		return
	}
	if stored != req.URL {
		writeErr(w, http.StatusConflict, "CODE_TAKEN", "This code is already taken.")
		return
	}
	s.cache.Add(code, stored)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]string{"shortUrl": "t/" + code})
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.ToLower(r.PathValue("code"))
	notFound := func() { writeErr(w, http.StatusNotFound, "NOT_FOUND", "Short link not found.") }
	if !validCode(code) {
		notFound()
		return
	}
	target, ok := s.cache.Get(code)
	if !ok {
		var err error
		target, ok, err = s.store.Get(r.Context(), code)
		if err != nil {
			log.Printf("get %q: %v", code, err)
			writeErr(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Service temporarily unavailable; please retry.")
			return
		}
		if !ok {
			notFound()
			return
		}
		s.cache.Add(code, target)
	}
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusMovedPermanently)
}
