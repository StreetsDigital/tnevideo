package middleware

import (
	"context"
	"testing"
)

func TestAnonymizeIPForLogging_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard IPv4", "192.168.1.100", "192.168.1.0"},
		{"already anonymized", "192.168.1.0", "192.168.1.0"},
		{"loopback", "127.0.0.1", "127.0.0.0"},
		{"public IP", "8.8.8.8", "8.8.8.0"},
		{"empty", "", "[no-ip]"},
		{"invalid", "not-an-ip", "[invalid-ip]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeIPForLogging(tt.input)
			if result != tt.expected {
				t.Errorf("AnonymizeIPForLogging(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnonymizeIPForLogging_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:db8:85a3::"},
		{"compressed IPv6", "2001:db8:85a3::8a2e:370:7334", "2001:db8:85a3::"},
		{"loopback IPv6", "::1", "::"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeIPForLogging(tt.input)
			if result != tt.expected {
				t.Errorf("AnonymizeIPForLogging(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnonymizeUserAgentForLogging(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "[no-ua]"},
		{"short UA", "Mozilla/5.0", "Mozilla/5.0"},
		{"long UA truncated", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWeb..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnonymizeUserAgentForLogging(tt.input)
			if result != tt.expected {
				t.Errorf("AnonymizeUserAgentForLogging(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPrivacyContext(t *testing.T) {
	ctx := context.Background()

	// Test defaults (no context set)
	if GDPRApplies(ctx) {
		t.Error("GDPRApplies should be false by default")
	}
	if GDPRConsentValidated(ctx) {
		t.Error("GDPRConsentValidated should be false by default")
	}
	if CCPAOptOut(ctx) {
		t.Error("CCPAOptOut should be false by default")
	}
	if GetConsentString(ctx) != "" {
		t.Error("GetConsentString should be empty by default")
	}

	// Test with context set
	ctx = SetPrivacyContext(ctx, true, true, false, "test-consent")
	if !GDPRApplies(ctx) {
		t.Error("GDPRApplies should be true")
	}
	if !GDPRConsentValidated(ctx) {
		t.Error("GDPRConsentValidated should be true")
	}
	if CCPAOptOut(ctx) {
		t.Error("CCPAOptOut should be false")
	}
	if GetConsentString(ctx) != "test-consent" {
		t.Errorf("GetConsentString should be 'test-consent', got %q", GetConsentString(ctx))
	}
}

func TestShouldCollectPII(t *testing.T) {
	tests := []struct {
		name           string
		gdprApplies    bool
		gdprConsented  bool
		ccpaOptOut     bool
		expectedResult bool
	}{
		{"no regulation", false, false, false, true},
		{"GDPR applies with consent", true, true, false, true},
		{"GDPR applies without consent", true, false, false, false},
		{"CCPA opt-out", false, false, true, false},
		{"GDPR and CCPA opt-out", true, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SetPrivacyContext(context.Background(), tt.gdprApplies, tt.gdprConsented, tt.ccpaOptOut, "")
			result := ShouldCollectPII(ctx)
			if result != tt.expectedResult {
				t.Errorf("ShouldCollectPII() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
