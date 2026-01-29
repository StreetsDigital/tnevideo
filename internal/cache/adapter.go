package cache

import (
	"context"

	"github.com/thenexusengine/tne_springwire/internal/exchange"
)

// ExchangeAdapter wraps a cache Store to implement exchange.CacheStore
type ExchangeAdapter struct {
	store *Store
}

// NewExchangeAdapter creates an adapter that bridges cache.Store to exchange.CacheStore
func NewExchangeAdapter(store *Store) *ExchangeAdapter {
	return &ExchangeAdapter{store: store}
}

// Put stores entries and returns UUIDs
func (a *ExchangeAdapter) Put(ctx context.Context, req *exchange.CachePutRequest) (*exchange.CachePutResponse, error) {
	puts := make([]PutObject, len(req.Puts))
	for i, p := range req.Puts {
		puts[i] = PutObject{
			Type:  p.Type,
			Value: p.Value,
		}
	}

	resp, err := a.store.Put(ctx, &PutRequest{Puts: puts})
	if err != nil {
		return nil, err
	}

	uuids := make([]string, len(resp.Responses))
	for i, r := range resp.Responses {
		uuids[i] = r.UUID
	}

	return &exchange.CachePutResponse{UUIDs: uuids}, nil
}

// CacheURL returns the full URL for a cached entry
func (a *ExchangeAdapter) CacheURL(uuid string) string {
	return a.store.CacheURL(uuid)
}

// CacheHost returns the cache hostname
func (a *ExchangeAdapter) CacheHost() string {
	return a.store.CacheHost()
}

// CachePath returns the cache path
func (a *ExchangeAdapter) CachePath() string {
	return a.store.CachePath()
}
