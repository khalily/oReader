# Bugs Found and Fixed During E2E Testing

## Bug #1: Login Response Missing CSRF Token

**Issue**: The Login endpoint was not returning `csrf_token` in the response body, causing inconsistency with the Register endpoint.

**Location**: `internal/handler/auth.go`, lines 250-260

**Root Cause**: The `Login` function was calling `setAuthCookies()` which sends the response with CSRF token, but then also tried to send a duplicate response without the CSRF token.

**Fix**: Removed the duplicate `c.JSON()` call in the `Login` function (lines 253-259), allowing `setAuthCookies()` to handle the complete response.

**Files Changed**:
- `internal/handler/auth.go`

**Impact**: Users can now extract the CSRF token from login responses, enabling proper CSRF protection for subsequent requests.

## Bug #2: Refresh Token Response Missing CSRF Token

**Issue**: The Refresh endpoint had the same issue as Login - duplicate response sending.

**Location**: `internal/handler/auth.go`, lines 386-394

**Root Cause**: Similar to Bug #1, the `Refresh` function was calling `setAuthCookies()` and then trying to send another response.

**Fix**: Removed the duplicate `c.JSON()` call in the `Refresh` function (lines 388-394), allowing `setAuthCookies()` to handle the complete response.

**Files Changed**:
- `internal/handler/auth.go`

**Impact**: Token refresh now properly returns CSRF token for continued authenticated requests.

## Summary

Both bugs were related to the same pattern: attempting to send multiple HTTP responses. The `setAuthCookies()` helper method is designed to send the complete response including the CSRF token, but the calling functions were also trying to send responses. This resulted in:

1. First response (from `setAuthCookies`) with CSRF token being ignored/overwritten
2. Second response (from caller) without CSRF token being sent
3. Clients unable to extract CSRF token for subsequent requests

The fix ensures all authentication endpoints (Register, Login, Refresh) consistently return the CSRF token in the response body.
