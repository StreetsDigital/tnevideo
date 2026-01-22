package identity

import (
	"context"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// AsyncRequest handles async identity resolution
type AsyncRequest struct {
	module           *Module
	ctx              context.Context
	cancel           context.CancelFunc
	Done             chan struct{}
	IdentityResponse *IdentityResponse
	Err              error
}

// NewAsyncRequest creates a new async request context
func (m *Module) NewAsyncRequest(r *http.Request) *AsyncRequest {
	ctx, cancel := context.WithCancel(r.Context())
	return &AsyncRequest{
		module: m,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Cancel cancels the async request
func (ar *AsyncRequest) Cancel() {
	ar.cancel()
}

// fetchIdentityAsync starts an async identity resolution request
func (ar *AsyncRequest) fetchIdentityAsync(bidRequest *openrtb2.BidRequest) {
	ar.Done = make(chan struct{})

	go func() {
		defer close(ar.Done)

		identityResp, err := ar.module.fetchIdentity(ar.ctx, bidRequest)
		ar.IdentityResponse = identityResp
		ar.Err = err
	}()
}
