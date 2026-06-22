package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const adminSessionCookieName = "agent_imageflow_admin"

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminSessionResponse struct {
	Authenticated bool       `json:"authenticated"`
	Username      string     `json:"username,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Configured    bool       `json:"configured"`
}

type adminSessionClaims struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

func (s *Server) handleAdminSessionRoute(w http.ResponseWriter, r *http.Request, parts []string) bool {
	switch {
	case r.Method == http.MethodPost && match(parts, "api", "admin", "login"):
		s.handleAdminLogin(w, r)
		return true
	case r.Method == http.MethodGet && match(parts, "api", "admin", "me"):
		s.handleAdminMe(w, r)
		return true
	case r.Method == http.MethodPost && match(parts, "api", "admin", "logout"):
		s.handleAdminLogout(w, r)
		return true
	default:
		return false
	}
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if !s.adminConfigured() {
		writeError(w, http.StatusServiceUnavailable, "admin_not_configured", "admin login is not configured")
		return
	}
	defer r.Body.Close()
	var req adminLoginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !s.verifyAdminCredentials(req.Username, req.Password) {
		writeUnauthorized(w, "admin_login_invalid", "admin username or password is invalid", false)
		return
	}
	expiresAt := time.Now().Add(s.adminSessionTTL())
	http.SetCookie(w, s.newAdminSessionCookie(strings.TrimSpace(req.Username), expiresAt))
	writeJSON(w, http.StatusOK, adminSessionResponse{
		Authenticated: true,
		Username:      strings.TrimSpace(req.Username),
		ExpiresAt:     &expiresAt,
		Configured:    true,
	})
}

func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	if !s.adminConfigured() {
		writeError(w, http.StatusServiceUnavailable, "admin_not_configured", "admin login is not configured")
		return
	}
	username, ok := s.adminSessionUsername(r)
	if !ok {
		writeUnauthorized(w, "admin_session_invalid", "admin session is missing or expired", false)
		return
	}
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		writeUnauthorized(w, "admin_session_invalid", "admin session is missing or expired", false)
		return
	}
	claims, ok := s.parseAdminSessionToken(cookie.Value)
	if !ok {
		writeUnauthorized(w, "admin_session_invalid", "admin session is missing or expired", false)
		return
	}
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	writeJSON(w, http.StatusOK, adminSessionResponse{
		Authenticated: true,
		Username:      username,
		ExpiresAt:     &expiresAt,
		Configured:    true,
	})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, s.clearAdminSessionCookie())
	writeJSON(w, http.StatusOK, adminSessionResponse{
		Authenticated: false,
		Configured:    s.adminConfigured(),
	})
}

func (s *Server) adminConfigured() bool {
	return strings.TrimSpace(s.options.AdminUsername) != "" && strings.TrimSpace(s.options.AdminPassword) != ""
}

func (s *Server) adminSessionTTL() time.Duration {
	if s.options.AdminSessionTTL > 0 {
		return s.options.AdminSessionTTL
	}
	return 12 * time.Hour
}

func (s *Server) verifyAdminCredentials(username, password string) bool {
	if !s.adminConfigured() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte(strings.TrimSpace(s.options.AdminUsername))) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(s.options.AdminPassword)) == 1
}

func (s *Server) adminSessionUsername(r *http.Request) (string, bool) {
	if !s.adminConfigured() {
		return "", false
	}
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		return "", false
	}
	claims, ok := s.parseAdminSessionToken(cookie.Value)
	if !ok {
		return "", false
	}
	username := strings.TrimSpace(s.options.AdminUsername)
	if subtle.ConstantTimeCompare([]byte(claims.Subject), []byte(username)) != 1 {
		return "", false
	}
	if time.Now().Unix() >= claims.ExpiresAt {
		return "", false
	}
	return username, true
}

func (s *Server) newAdminSessionCookie(username string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    s.signAdminSessionClaims(adminSessionClaims{Subject: username, ExpiresAt: expiresAt.Unix()}),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *Server) clearAdminSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *Server) signAdminSessionClaims(claims adminSessionClaims) string {
	raw, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	signature := s.signAdminSessionPayload(payload)
	return payload + "." + signature
}

func (s *Server) parseAdminSessionToken(token string) (adminSessionClaims, bool) {
	payload, signature, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || payload == "" || signature == "" {
		return adminSessionClaims{}, false
	}
	expected := s.signAdminSessionPayload(payload)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return adminSessionClaims{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return adminSessionClaims{}, false
	}
	var claims adminSessionClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return adminSessionClaims{}, false
	}
	return claims, strings.TrimSpace(claims.Subject) != "" && claims.ExpiresAt > 0
}

func (s *Server) signAdminSessionPayload(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.adminSessionSecret()))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) adminSessionSecret() string {
	if secret := strings.TrimSpace(s.options.AdminSessionSecret); secret != "" {
		return secret
	}
	return s.options.AdminPassword
}

func routeAllowsAdminSession(parts []string, method string) bool {
	isRead := method == http.MethodGet || method == http.MethodHead
	switch {
	case isRead && match(parts, "api", "admin", "assets", "recent"):
		return true
	case isRead && match(parts, "api", "admin", "runtime-status"):
		return true
	case (isRead || method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete) &&
		(match(parts, "api", "workspaces") ||
			match(parts, "api", "workspaces", "*") ||
			match(parts, "api", "workspaces", "*", "projects") ||
			match(parts, "api", "workspaces", "*", "projects", "*") ||
			match(parts, "api", "workspaces", "*", "projects", "*", "campaigns") ||
			match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*")):
		return true
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*", "promote-asset"):
		return true
	case (isRead || method == http.MethodPost) && match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		return true
	case (isRead || method == http.MethodPost) && match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		return true
	case isRead && (match(parts, "api", "tasks", "*") ||
		match(parts, "api", "tasks", "*", "attempts") ||
		match(parts, "api", "projects", "*", "campaigns", "*", "assets") ||
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-progress") ||
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-summary") ||
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-manifest") ||
		match(parts, "api", "assets", "*") ||
		match(parts, "api", "assets", "*", "metadata") ||
		match(parts, "api", "assets", "*", "original") ||
		match(parts, "api", "assets", "*", "thumbnail")):
		return true
	case method == http.MethodPost && (match(parts, "api", "assets", "*", "approve") ||
		match(parts, "api", "assets", "*", "reject") ||
		match(parts, "api", "projects", "*", "campaigns", "*", "scene-regenerations")):
		return true
	default:
		return false
	}
}
