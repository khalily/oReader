package cookie

import (
	"net/http"
)

// Config holds cookie configuration
type Config struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string
	Path     string
}

// DefaultConfig returns default cookie configuration
func DefaultConfig(isProduction bool) *Config {
	sameSite := http.SameSiteStrictMode
	if !isProduction {
		sameSite = http.SameSiteLaxMode
	}
	return &Config{
		Secure:   isProduction,
		SameSite: sameSite,
		Path:     "/",
	}
}

// SetAccessToken sets the access token cookie
func SetAccessToken(w http.ResponseWriter, token string, csrfToken string, cfg *Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   900, // 15 minutes
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})

	// Also set CSRF token cookie (readable by JS)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   900, // 15 minutes
		Secure:   cfg.Secure,
		HttpOnly: false, // Readable by JS
		SameSite: cfg.SameSite,
	})
}

// SetRefreshToken sets the refresh token cookie
func SetRefreshToken(w http.ResponseWriter, token string, cfg *Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/v1/auth/refresh", // Only sent to refresh endpoint
		Domain:   cfg.Domain,
		MaxAge:   604800, // 7 days
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})
}

// ClearTokens clears all auth cookies
func ClearTokens(w http.ResponseWriter, cfg *Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   -1,
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth/refresh",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   -1,
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
	})
}

// GetAccessToken retrieves the access token from the request
func GetAccessToken(r *http.Request) string {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetRefreshToken retrieves the refresh token from the request
func GetRefreshToken(r *http.Request) string {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetCSRFToken retrieves the CSRF token from the request
func GetCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}
