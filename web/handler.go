package web

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/hvsio/ma-todo-res/manager"
	"github.com/hvsio/ma-todo-res/store"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	store   *store.Store
	manager *manager.Manager
	tmpl    *template.Template
	static  fs.FS
}

// PageData is passed to the index template.
type PageData struct {
	Posts      []store.StoredPost
	Hashtags   []string
	LastUpdate time.Time
}

func NewHandler(s *store.Store, m *manager.Manager, templateFS fs.FS, staticFS fs.FS) (*Handler, error) {
	funcMap := template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) == 0 {
				return ""
			}
			runes := []rune(s)
			if len(runes) <= n {
				return s
			}
			return string(runes[:n]) + "…"
		},
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "index.html")
	if err != nil {
		return nil, err
	}
	return &Handler{store: s, manager: m, tmpl: tmpl, static: staticFS}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.handlePage)
	mux.HandleFunc("GET /api/posts", h.handleAPIPosts)
	mux.HandleFunc("GET /api/hashtags", h.handleAPIHashtagsList)
	mux.HandleFunc("POST /api/hashtags", h.handleAPIHashtagsAdd)
	mux.HandleFunc("DELETE /api/hashtags/{hashtag}", h.handleAPIHashtagsRemove)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(h.static))))
}

func (h *Handler) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := PageData{
		Posts:      h.store.List(),
		Hashtags:   h.manager.List(),
		LastUpdate: time.Now(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleAPIPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	posts := h.store.List()
	if posts == nil {
		posts = []store.StoredPost{}
	}
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, `{"error":"encode failed"}`, http.StatusInternalServerError)
	}
}

func (h *Handler) handleAPIHashtagsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tags := h.manager.List()
	if tags == nil {
		tags = []string{}
	}
	json.NewEncoder(w).Encode(tags)
}

func (h *Handler) handleAPIHashtagsAdd(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hashtag string `json:"hashtag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	tag := strings.TrimSpace(strings.TrimPrefix(body.Hashtag, "#"))
	if tag == "" {
		http.Error(w, `{"error":"hashtag is required"}`, http.StatusBadRequest)
		return
	}
	if err := h.manager.Add(tag); err != nil {
		// Already running — treat as idempotent 409.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) handleAPIHashtagsRemove(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("hashtag")
	if tag == "" {
		http.Error(w, `{"error":"hashtag is required"}`, http.StatusBadRequest)
		return
	}
	h.manager.Remove(tag)
	w.WriteHeader(http.StatusNoContent)
}
