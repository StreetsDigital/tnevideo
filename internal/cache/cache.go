// Package cache implements Prebid Cache for storing and retrieving bid creatives.
// It supports both VAST XML (for video) and JSON (for banner/native) entries,
// backed by Redis with configurable TTL.
package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	pkgredis "github.com/thenexusengine/tne_springwire/pkg/redis"
)

const (
	// DefaultTTL for cached entries (15 minutes, per Prebid spec)
	DefaultTTL = 15 * time.Minute

	// MaxEntries per single PUT request
	MaxEntries = 100

	// MaxValueSize per entry (512KB)
	MaxValueSize = 512 * 1024

	// Redis key prefix
	keyPrefix = "prebid_cache:"
)

// PutObject represents a single cache entry in a PUT request
type PutObject struct {
	Type  string `json:"type"`  // "xml" or "json"
	Value any    `json:"value"` // The content to cache
	TTL   int    `json:"ttlseconds,omitempty"`
}

// PutRequest is the body for POST /cache
type PutRequest struct {
	Puts []PutObject `json:"puts"`
}

// PutResponseEntry is a single UUID in the PUT response
type PutResponseEntry struct {
	UUID string `json:"uuid"`
}

// PutResponse is returned from POST /cache
type PutResponse struct {
	Responses []PutResponseEntry `json:"responses"`
}

// CachedEntry is what's stored in Redis
type CachedEntry struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Created int64  `json:"created"`
}

// Store provides Prebid Cache storage backed by Redis
type Store struct {
	redis      *pkgredis.Client
	defaultTTL time.Duration
	hostURL    string // For constructing cache URLs
}

// NewStore creates a new cache store
func NewStore(redis *pkgredis.Client, hostURL string) *Store {
	return &Store{
		redis:      redis,
		defaultTTL: DefaultTTL,
		hostURL:    hostURL,
	}
}

// Put stores one or more entries and returns their UUIDs
func (s *Store) Put(ctx context.Context, req *PutRequest) (*PutResponse, error) {
	if len(req.Puts) == 0 {
		return &PutResponse{Responses: []PutResponseEntry{}}, nil
	}

	if len(req.Puts) > MaxEntries {
		return nil, fmt.Errorf("too many entries: %d (max %d)", len(req.Puts), MaxEntries)
	}

	resp := &PutResponse{
		Responses: make([]PutResponseEntry, len(req.Puts)),
	}

	for i, obj := range req.Puts {
		uuid, err := s.putSingle(ctx, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to cache entry %d: %w", i, err)
		}
		resp.Responses[i] = PutResponseEntry{UUID: uuid}
	}

	return resp, nil
}

// Get retrieves a cached entry by UUID
func (s *Store) Get(ctx context.Context, uuid string) (*CachedEntry, error) {
	key := keyPrefix + uuid

	data, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}
	if data == "" {
		return nil, nil // Not found
	}

	var entry CachedEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, fmt.Errorf("corrupt cache entry: %w", err)
	}

	return &entry, nil
}

// CacheURL returns the full URL for retrieving a cached entry
func (s *Store) CacheURL(uuid string) string {
	return fmt.Sprintf("%s/cache?uuid=%s", s.hostURL, uuid)
}

// CacheHost returns the hostname portion of the cache URL
func (s *Store) CacheHost() string {
	return s.hostURL
}

// CachePath returns the path portion of the cache endpoint
func (s *Store) CachePath() string {
	return "/cache"
}

func (s *Store) putSingle(ctx context.Context, obj PutObject) (string, error) {
	// Serialize the value to string
	var valueStr string
	switch v := obj.Value.(type) {
	case string:
		valueStr = v
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to serialize value: %w", err)
		}
		valueStr = string(bytes)
	}

	if len(valueStr) > MaxValueSize {
		return "", fmt.Errorf("value too large: %d bytes (max %d)", len(valueStr), MaxValueSize)
	}

	// Validate type
	if obj.Type != "xml" && obj.Type != "json" {
		return "", fmt.Errorf("invalid type %q: must be \"xml\" or \"json\"", obj.Type)
	}

	uuid := generateUUID()

	entry := CachedEntry{
		Type:    obj.Type,
		Value:   valueStr,
		Created: time.Now().Unix(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entry: %w", err)
	}

	ttl := s.defaultTTL
	if obj.TTL > 0 {
		ttl = time.Duration(obj.TTL) * time.Second
	}

	key := keyPrefix + uuid
	if err := s.redis.SetEx(ctx, key, string(data), ttl); err != nil {
		return "", fmt.Errorf("redis SET failed: %w", err)
	}

	return uuid, nil
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// Format as UUID v4: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 1
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
