package endpoints

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/thenexusengine/tne_springwire/internal/cache"
	"github.com/thenexusengine/tne_springwire/pkg/logger"
)

// CacheHandler handles Prebid Cache requests (POST to store, GET to retrieve).
// This implements the Prebid Cache protocol that Prebid.js expects:
//   - POST /cache  -> store VAST XML or JSON, return UUIDs
//   - GET  /cache?uuid=<id> -> retrieve cached content
type CacheHandler struct {
	store *cache.Store
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(store *cache.Store) *CacheHandler {
	return &CacheHandler{store: store}
}

// ServeHTTP routes to the appropriate handler based on method
func (h *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permissive CORS for ad serving (cross-origin requests are expected)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handlePut(w, r)
	case http.MethodGet:
		h.handleGet(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePut stores entries in the cache
func (h *CacheHandler) handlePut(w http.ResponseWriter, r *http.Request) {
	log := logger.Log

	body, err := io.ReadAll(io.LimitReader(r.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		log.Warn().Err(err).Msg("cache: failed to read request body")
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req cache.PutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Warn().Err(err).Msg("cache: invalid JSON")
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := h.store.Put(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Msg("cache: failed to store entries")
		http.Error(w, "cache storage failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("cache: failed to encode response")
	}
}

// handleGet retrieves an entry from the cache by UUID
func (h *CacheHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	log := logger.Log

	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		http.Error(w, "missing uuid parameter", http.StatusBadRequest)
		return
	}

	// Basic input validation: UUIDs should be hex+dashes, max 36 chars
	if len(uuid) > 36 {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	entry, err := h.store.Get(r.Context(), uuid)
	if err != nil {
		log.Error().Err(err).Str("uuid", uuid).Msg("cache: retrieval error")
		http.Error(w, "cache retrieval failed", http.StatusInternalServerError)
		return
	}

	if entry == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Return with appropriate content type per Prebid Cache spec
	switch entry.Type {
	case "xml":
		w.Header().Set("Content-Type", "application/xml")
	case "json":
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "text/plain")
	}

	w.Write([]byte(entry.Value))
}
