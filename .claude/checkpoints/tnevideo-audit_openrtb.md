# OpenRTB 2.5 Protocol Compliance Audit
**Project:** tnevideo
**Date:** 2026-01-26
**Status:** ANALYSIS COMPLETE

## Files Examined

### Core OpenRTB Implementation
- `/Users/andrewstreets/tnevideo/tnevideo/internal/openrtb/request.go` - Data models
- `/Users/andrewstreets/tnevideo/tnevideo/internal/openrtb/response.go` - Response models
- `/Users/andrewstreets/tnevideo/tnevideo/internal/exchange/exchange.go` - Auction logic (2300+ lines)
- `/Users/andrewstreets/tnevideo/tnevideo/internal/endpoints/auction.go` - Request handling
- `/Users/andrewstreets/tnevideo/tnevideo/internal/adapters/adapter.go` - Bidder framework
- `/Users/andrewstreets/tnevideo/tnevideo/internal/adapters/helpers.go` - Adapter utilities
- `/Users/andrewstreets/tnevideo/tnevideo/internal/adapters/ortb/ortb.go` - Generic ORTB adapter

### Adapters Reviewed
- appnexus, rubicon, pubmatic, ix, openx, criteo, demo
- conversant, beachfront, triplelift, sharethrough, etc.

## VIOLATIONS FOUND

### CRITICAL - MUST FIX

#### 1. Missing Media Type Exclusivity Validation (Banner + Video in same Imp)
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/exchange/exchange.go`
**Lines:** 1114-1138
**Issue:** OpenRTB 2.5 Section 3.2.4 states that while an impression CAN have multiple media types, the bid response must specify which type it's for. Currently no validation ensures bid type matches impression's offered types.
**Risk:** Bidders could return video bids for banner-only impressions.

#### 2. Missing ADomain Field Validation
**Files:** All adapters, exchange.go validateBid()
**Lines:** exchange.go:491-587
**Issue:** OpenRTB 2.5 recommends ADomain for brand safety. No validation that:
- ADomain is present (recommended for transparency)
- ADomain entries are valid domains (not arbitrary strings)
- ADomain doesn't contain blocked advertisers from request.badv
**Risk:** Brand safety bypass, blocked advertiser creatives served.

#### 3. Missing NURL Format Validation  
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/exchange/exchange.go`
**Lines:** 575-584
**Issue:** Only checks that AdM or NURL exists, but doesn't validate NURL is a valid URL format with proper scheme (http/https).
**Risk:** Malformed/malicious URLs could be returned to clients.

### HIGH - SHOULD FIX

#### 4. Response ID Validation - Partial Implementation
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/exchange/exchange.go`
**Lines:** 2067-2075
**Status:** IMPLEMENTED but incomplete
**Issue:** Validates ResponseID matches RequestID when ResponseID is non-empty. Per OpenRTB 2.5 Section 4.2.1, BidResponse.id is REQUIRED and must echo BidRequest.id. Should reject responses with empty ID.

#### 5. Currency Validation - Partial Implementation
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/exchange/exchange.go`
**Lines:** 2077-2098
**Status:** IMPLEMENTED 
**Good:** Validates response currency matches exchange currency, defaults empty to USD.
**Gap:** No validation that bid.cur exists in request.cur allowlist.

#### 6. Missing Impression Media Type Mutual Exclusivity Check in Bids
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/adapters/helpers.go`
**Lines:** 94-130 (GetBidType, GetBidTypeFromMap)
**Issue:** Determines bid type by checking which media type exists in impression (Video > Native > Audio > Banner priority). Doesn't verify the bid actually matches what bidder claimed.

### MEDIUM - FIX WHEN POSSIBLE

#### 7. Missing Bid.BURL Validation
**Issue:** Billing URL (burl) should be a valid URL if present.
**Location:** Not validated anywhere.

#### 8. Missing W/H Validation for Banner Bids
**Issue:** When impression specifies required banner sizes, bid dimensions should match.
**Location:** exchange.go validateBid() doesn't check dimensions.

#### 9. No Deal ID Validation
**Issue:** If bid specifies dealid, should validate it exists in impression.pmp.deals
**Location:** Not validated.

## EXISTING GOOD VALIDATIONS

1. **Request ID required** - exchange.go:1088-1089, endpoints/auction.go:176-178
2. **Impression required** - exchange.go:1091-1093, endpoints/auction.go:179-181
3. **Impression ID required & unique** - exchange.go:1114-1124
4. **Site/App mutual exclusivity** - exchange.go:1104-1112
5. **Media type required per impression** - exchange.go:1126-1128
6. **Banner dimensions validation** - exchange.go:1131-1137
7. **Bid ID required** - exchange.go:496-502
8. **Bid ImpID required** - exchange.go:504-512
9. **ImpID exists in request** - exchange.go:514-523
10. **Price validation** (non-negative, not NaN/Inf, max reasonable) - exchange.go:525-553
11. **Floor price enforcement** - exchange.go:566-573
12. **AdM or NURL required** - exchange.go:575-584
13. **Duplicate bid ID detection** - exchange.go:1377-1389
14. **Response ID matching** - exchange.go:2067-2075
15. **Currency matching** - exchange.go:2077-2098
16. **TMax bounds validation** - exchange.go:1156-1168

## NEXT STEPS

1. Implement ADomain validation in validateBid()
2. Add NURL URL format validation
3. Add bid type vs impression media type consistency check
4. Require BidResponse.ID (not just validate when present)
5. Add blocked advertiser (badv) check against bid.adomain
6. Add W/H dimension validation for banner bids
7. Add deal ID validation against PMP deals

