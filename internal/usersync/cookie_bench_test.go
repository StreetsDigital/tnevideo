package usersync

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// createLargeCookie creates a cookie with many UIDs that exceeds MaxCookieSize
func createLargeCookie(numUIDs int) *Cookie {
	cookie := NewCookie()
	
	for i := 0; i < numUIDs; i++ {
		bidder := "bidder"
		if i < 10 {
			bidder = "bidder0" + string(rune('0'+i))
		} else {
			bidder = "bidder" + string(rune('0'+(i/10))) + string(rune('0'+(i%10)))
		}
		
		cookie.SetUID(bidder, "uid_"+bidder+"_very_long_user_id_string_12345678901234567890")
	}
	
	return cookie
}

// BenchmarkTrimToFit_Old simulates the old O(n²) approach
func BenchmarkTrimToFit_Old(b *testing.B) {
	b.Run("10UIDs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cookie := createLargeCookie(10)
			b.StartTimer()
			
			// Old approach: marshal in loop until fits
			for len(cookie.UIDs) > 0 {
				data, err := json.Marshal(cookie)
				if err != nil {
					break
				}
				encoded := base64.URLEncoding.EncodeToString(data)
				if len(encoded) <= MaxCookieSize {
					break
				}
				
				// Find and remove oldest
				var oldestBidder string
				var oldestTime time.Time
				for bidder, uid := range cookie.UIDs {
					if oldestBidder == "" || uid.Expires.Before(oldestTime) {
						oldestBidder = bidder
						oldestTime = uid.Expires
					}
				}
				if oldestBidder != "" {
					delete(cookie.UIDs, oldestBidder)
				}
			}
		}
	})

	b.Run("50UIDs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cookie := createLargeCookie(50)
			b.StartTimer()
			
			for len(cookie.UIDs) > 0 {
				data, err := json.Marshal(cookie)
				if err != nil {
					break
				}
				encoded := base64.URLEncoding.EncodeToString(data)
				if len(encoded) <= MaxCookieSize {
					break
				}
				
				var oldestBidder string
				var oldestTime time.Time
				for bidder, uid := range cookie.UIDs {
					if oldestBidder == "" || uid.Expires.Before(oldestTime) {
						oldestBidder = bidder
						oldestTime = uid.Expires
					}
				}
				if oldestBidder != "" {
					delete(cookie.UIDs, oldestBidder)
				}
			}
		}
	})
}

// BenchmarkTrimToFit_New tests the optimized binary search approach
func BenchmarkTrimToFit_New(b *testing.B) {
	b.Run("10UIDs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cookie := createLargeCookie(10)
			b.StartTimer()
			
			cookie.trimToFit()
		}
	})

	b.Run("50UIDs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cookie := createLargeCookie(50)
			b.StartTimer()
			
			cookie.trimToFit()
		}
	})

	b.Run("100UIDs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			cookie := createLargeCookie(100)
			b.StartTimer()
			
			cookie.trimToFit()
		}
	})
}

// BenchmarkCookieEncode tests full encode path with trimming
func BenchmarkCookieEncode(b *testing.B) {
	b.Run("SmallCookie_NoTrim", func(b *testing.B) {
		cookie := createLargeCookie(5)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_, _ = cookie.ToHTTPCookie("example.com")
		}
	})

	b.Run("LargeCookie_WithTrim", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			testCookie := createLargeCookie(50)
			b.StartTimer()
			
			_, _ = testCookie.ToHTTPCookie("example.com")
		}
	})
}

// BenchmarkMarshalOperations measures marshal overhead
func BenchmarkMarshalOperations(b *testing.B) {
	cookie := createLargeCookie(20)
	
	b.Run("SingleMarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(cookie)
		}
	})

	b.Run("MarshalAndEncode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(cookie)
			_ = base64.URLEncoding.EncodeToString(data)
		}
	})
}
