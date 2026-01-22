package pauseads

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// Handler provides HTTP endpoint handling for pause ad requests
type Handler struct {
	// AuctionRunner is a function that runs the bid auction
	// In a real implementation, this would integrate with the exchange module
	AuctionRunner func(*openrtb2.BidRequest) (*openrtb2.BidResponse, error)
}

// NewHandler creates a new pause ad handler
func NewHandler(auctionRunner func(*openrtb2.BidRequest) (*openrtb2.BidResponse, error)) *Handler {
	return &Handler{
		AuctionRunner: auctionRunner,
	}
}

// HandlePauseAdRequest handles HTTP requests for pause ads
func (h *Handler) HandlePauseAdRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the OpenRTB bid request from the HTTP request body
	var bidRequest openrtb2.BidRequest
	if err := json.NewDecoder(r.Body).Decode(&bidRequest); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Detect pause event
	pauseReq, err := DetectPauseEvent(&bidRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to detect pause event: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the pause ad request
	if err := IsValidPauseAdRequest(pauseReq); err != nil {
		http.Error(w, fmt.Sprintf("invalid pause ad request: %v", err), http.StatusBadRequest)
		return
	}

	// Handle resume events (no ad to serve)
	if pauseReq.State == StateResumed {
		if pauseReq.SessionID != "" {
			if err := HandleResume(pauseReq.SessionID); err != nil {
				http.Error(w, fmt.Sprintf("failed to handle resume: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// Return empty response for resume
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&PauseAdResponse{
			Format: pauseReq.Format,
		})
		return
	}

	// Run the auction if we have an auction runner
	var bidResponse *openrtb2.BidResponse
	if h.AuctionRunner != nil {
		bidResponse, err = h.AuctionRunner(&bidRequest)
		if err != nil {
			http.Error(w, fmt.Sprintf("auction failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Serve the pause ad
	pauseResp, err := ServePauseAd(pauseReq, bidResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to serve pause ad: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the pause ad response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(pauseResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.HandlePauseAdRequest(w, r)
}
