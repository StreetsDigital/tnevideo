package exchange

import (
	"fmt"
	"net/http"
	"time"

	"github.com/thenexusengine/tne_springwire/internal/ctv"
	"github.com/thenexusengine/tne_springwire/internal/openrtb"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// VASTResponseBuilder builds VAST responses from bid responses
type VASTResponseBuilder struct {
	trackingBaseURL string
	version         string
}

// NewVASTResponseBuilder creates a new VAST response builder
func NewVASTResponseBuilder(trackingBaseURL string) *VASTResponseBuilder {
	return &VASTResponseBuilder{
		trackingBaseURL: trackingBaseURL,
		version:         "4.0",
	}
}

// BuildVASTResponse creates a VAST response from a bid response
func (b *VASTResponseBuilder) BuildVASTResponse(bidReq *openrtb.BidRequest, bidResp *BidResponse) (*vast.VAST, error) {
	if bidResp == nil || len(bidResp.SeatBid) == 0 {
		return vast.CreateEmptyVAST(), nil
	}

	builder := vast.NewBuilder(b.version)

	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			// Extract video impression
			imp := findImpression(bidReq.Imp, bid.ImpID)
			if imp == nil || imp.Video == nil {
				continue
			}

			// Build ad
			builder.AddAd(bid.ID).
				WithInLine("TNEVideo", bid.AdID).
				WithImpression(fmt.Sprintf("%s/video/impression?bid_id=%s&bidder=%s", b.trackingBaseURL, bid.ID, seatBid.Seat)).
				WithError(fmt.Sprintf("%s/video/error?bid_id=%s&bidder=%s", b.trackingBaseURL, bid.ID, seatBid.Seat))

			// Add linear creative
			duration := time.Duration(imp.Video.MaxDuration) * time.Second
			if duration == 0 {
				duration = 30 * time.Second
			}

			linearBuilder := builder.WithLinearCreative(bid.ID+"-creative", duration)

			// Add media file from NURL or ADM
			mediaURL := bid.NURL
			if bid.AdM != "" {
				mediaURL = bid.AdM
			}

			// Determine video format
			mimeType := "video/mp4"
			if len(imp.Video.Mimes) > 0 {
				mimeType = imp.Video.Mimes[0]
			}

			linearBuilder.WithMediaFile(
				mediaURL,
				mimeType,
				imp.Video.W,
				imp.Video.H,
				vast.WithBitrate(imp.Video.MaxBitrate),
			)

			// Add tracking events
			linearBuilder.WithAllQuartileTracking(fmt.Sprintf("%s/video/event?bid_id=%s&bidder=%s", b.trackingBaseURL, bid.ID, seatBid.Seat))

			// Add skip offset for skippable ads
			if imp.Video.Skip != nil && *imp.Video.Skip == 1 {
				offset := "00:00:05"
				if imp.Video.SkipAfter > 0 {
					offset = fmt.Sprintf("00:00:%02d", imp.Video.SkipAfter)
				}
				linearBuilder.WithSkipOffset(offset)
			}

			linearBuilder.EndLinear().Done()
		}
	}

	return builder.Build()
}

// BidResponse represents a bid response (simplified)
type BidResponse struct {
	ID      string    `json:"id"`
	SeatBid []SeatBid `json:"seatbid"`
	Cur     string    `json:"cur"`
}

// SeatBid represents a seat bid
type SeatBid struct {
	Bid  []Bid  `json:"bid"`
	Seat string `json:"seat"`
}

// Bid represents a single bid
type Bid struct {
	ID    string  `json:"id"`
	ImpID string  `json:"impid"`
	Price float64 `json:"price"`
	NURL  string  `json:"nurl,omitempty"`
	AdM   string  `json:"adm,omitempty"`
	AdID  string  `json:"adid,omitempty"`
	W     int     `json:"w,omitempty"`
	H     int     `json:"h,omitempty"`
}

// findImpression finds an impression by ID
func findImpression(imps []openrtb.Imp, impID string) *openrtb.Imp {
	for i := range imps {
		if imps[i].ID == impID {
			return &imps[i]
		}
	}
	return nil
}

// VASTHandler handles VAST endpoint requests
type VASTHandler struct {
	builder  *VASTResponseBuilder
	exchange *Exchange
}

// Exchange is a placeholder for the actual exchange implementation
type Exchange interface {
	RunAuction(req *openrtb.BidRequest) (*BidResponse, error)
}

// NewVASTHandler creates a new VAST handler
func NewVASTHandler(exchange Exchange, trackingBaseURL string) *VASTHandler {
	return &VASTHandler{
		builder:  NewVASTResponseBuilder(trackingBaseURL),
		exchange: exchange,
	}
}

// ServeHTTP handles VAST endpoint requests
func (h *VASTHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse bid request from query params or body
	bidReq, err := parseBidRequest(r)
	if err != nil {
		writeVASTError(w, "Invalid request")
		return
	}

	// Detect CTV device
	if bidReq.Device != nil {
		deviceInfo := ctv.DetectDevice(bidReq.Device)
		if deviceInfo.IsCTV {
			// Apply CTV-specific optimizations
			applyCTVOptimizations(bidReq, deviceInfo)
		}
	}

	// Run auction
	bidResp, err := h.exchange.RunAuction(bidReq)
	if err != nil {
		writeVASTError(w, "Auction failed")
		return
	}

	// Build VAST response
	vastResp, err := h.builder.BuildVASTResponse(bidReq, bidResp)
	if err != nil {
		writeVASTError(w, "Failed to build VAST")
		return
	}

	// Marshal and write response
	data, err := vastResp.Marshal()
	if err != nil {
		writeVASTError(w, "Failed to serialize VAST")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

// parseBidRequest parses a bid request from an HTTP request
func parseBidRequest(r *http.Request) (*openrtb.BidRequest, error) {
	// Simplified implementation - would need full OpenRTB parsing
	return &openrtb.BidRequest{
		ID: r.URL.Query().Get("id"),
		Imp: []openrtb.Imp{
			{
				ID: "1",
				Video: &openrtb.Video{
					Mimes:       []string{"video/mp4"},
					MinDuration: 5,
					MaxDuration: 30,
					W:           1920,
					H:           1080,
				},
			},
		},
	}, nil
}

// applyCTVOptimizations applies CTV-specific optimizations to the bid request
func applyCTVOptimizations(bidReq *openrtb.BidRequest, deviceInfo *ctv.DeviceInfo) {
	caps := ctv.GetCapabilities(deviceInfo.Type)

	for i := range bidReq.Imp {
		if bidReq.Imp[i].Video != nil {
			// Limit bitrate based on device capabilities
			if bidReq.Imp[i].Video.MaxBitrate > caps.MaxBitrate {
				bidReq.Imp[i].Video.MaxBitrate = caps.MaxBitrate
			}

			// Filter VPAID if not supported
			if !caps.SupportsVPAID {
				// Remove VPAID APIs from request
				filtered := make([]int, 0)
				for _, api := range bidReq.Imp[i].Video.API {
					if api != 1 && api != 2 { // 1 = VPAID 1.0, 2 = VPAID 2.0
						filtered = append(filtered, api)
					}
				}
				bidReq.Imp[i].Video.API = filtered
			}
		}
	}
}

// writeVASTError writes a VAST error response
func writeVASTError(w http.ResponseWriter, message string) {
	v := &vast.VAST{
		Version: "4.0",
		Error:   message,
	}
	data, _ := v.Marshal()
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK) // VAST always returns 200
	w.Write(data)
}
