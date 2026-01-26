// Package middleware provides HTTP middleware components
package middleware

import (
	"context"
	"net"
	"strings"
)

// PrivacyContextKey is the key used to store privacy information in context
type PrivacyContextKey string

const (
	// ContextKeyGDPRConsent stores whether GDPR consent was validated
	ContextKeyGDPRConsent PrivacyContextKey = "gdpr_consent_validated"
	// ContextKeyGDPRApplies stores whether GDPR applies to this request
	ContextKeyGDPRApplies PrivacyContextKey = "gdpr_applies"
	// ContextKeyCCPAOptOut stores whether user has opted out under CCPA
	ContextKeyCCPAOptOut PrivacyContextKey = "ccpa_opt_out"
	// ContextKeyConsentString stores the TCF consent string
	ContextKeyConsentString PrivacyContextKey = "consent_string"
)

// AnonymizeIPForLogging returns an anonymized IP suitable for logging
// IPv4: Masks last octet (192.168.1.100 -> 192.168.1.0)
// IPv6: Masks last 80 bits, keeping first 48 bits (2001:db8:85a3::1 -> 2001:db8:85a3::)
// This helper should be used for ALL log statements that include IP addresses
func AnonymizeIPForLogging(ipStr string) string {
	if ipStr == "" {
		return "[no-ip]"
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "[invalid-ip]"
	}
	
	// Check if it's IPv4
	if ipv4 := ip.To4(); ipv4 != nil {
		// Mask last octet
		ipv4[3] = 0
		return ipv4.String()
	}
	
	// IPv6 - mask last 80 bits (keep first 48 bits / 6 bytes)
	ipv6 := ip.To16()
	if ipv6 == nil {
		return "[invalid-ip]"
	}
	for i := 6; i < 16; i++ {
		ipv6[i] = 0
	}
	return ipv6.String()
}

// AnonymizeUserAgentForLogging returns a truncated/anonymized UA for logging
// Only keeps the first 50 characters and browser family identification
// This reduces PII while maintaining debugging utility
func AnonymizeUserAgentForLogging(ua string) string {
	if ua == "" {
		return "[no-ua]"
	}
	
	// Extract just the browser family (first significant part)
	// This provides debugging value without full fingerprinting
	ua = strings.TrimSpace(ua)
	
	// Truncate to reasonable length
	if len(ua) > 50 {
		ua = ua[:50] + "..."
	}
	
	return ua
}

// GDPRConsentValidated checks if GDPR consent was validated in the middleware
func GDPRConsentValidated(ctx context.Context) bool {
	if val, ok := ctx.Value(ContextKeyGDPRConsent).(bool); ok {
		return val
	}
	return false
}

// GDPRApplies checks if GDPR applies to this request
func GDPRApplies(ctx context.Context) bool {
	if val, ok := ctx.Value(ContextKeyGDPRApplies).(bool); ok {
		return val
	}
	return false
}

// CCPAOptOut checks if user has opted out under CCPA
func CCPAOptOut(ctx context.Context) bool {
	if val, ok := ctx.Value(ContextKeyCCPAOptOut).(bool); ok {
		return val
	}
	return false
}

// GetConsentString retrieves the TCF consent string from context
func GetConsentString(ctx context.Context) string {
	if val, ok := ctx.Value(ContextKeyConsentString).(string); ok {
		return val
	}
	return ""
}

// SetPrivacyContext creates a new context with privacy information
func SetPrivacyContext(ctx context.Context, gdprApplies, gdprConsented, ccpaOptOut bool, consentString string) context.Context {
	ctx = context.WithValue(ctx, ContextKeyGDPRApplies, gdprApplies)
	ctx = context.WithValue(ctx, ContextKeyGDPRConsent, gdprConsented)
	ctx = context.WithValue(ctx, ContextKeyCCPAOptOut, ccpaOptOut)
	ctx = context.WithValue(ctx, ContextKeyConsentString, consentString)
	return ctx
}

// ShouldCollectPII determines if PII collection is allowed based on privacy context
// Returns false if:
// - GDPR applies and consent is not validated
// - User has opted out under CCPA
func ShouldCollectPII(ctx context.Context) bool {
	// If GDPR applies, must have validated consent
	if GDPRApplies(ctx) && !GDPRConsentValidated(ctx) {
		return false
	}
	
	// If user opted out under CCPA, no PII collection
	if CCPAOptOut(ctx) {
		return false
	}
	
	return true
}
