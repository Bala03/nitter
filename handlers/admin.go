package handlers

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"time"

	"github.com/zedeus/nitter/twitter"
)

// RegisterAdminRoutes sets up the admin panel routes
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin", h.adminAuth(h.handleAdmin))
	mux.HandleFunc("/admin/login", h.handleAdminLogin)
	mux.HandleFunc("/admin/add-session", h.adminAuth(h.handleAddSession))
	mux.HandleFunc("/admin/remove-session", h.adminAuth(h.handleRemoveSession))
	mux.HandleFunc("/admin/toggle-session", h.adminAuth(h.handleToggleSession))
	mux.HandleFunc("/admin/validate-session", h.adminAuth(h.handleValidateSession))
}

// adminAuth wraps a handler with admin authentication
func (h *Handler) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.cfg.AdminPassword == "" {
			h.renderError(w, "Admin panel is disabled. Set adminPassword in nitter.conf to enable it.", http.StatusForbidden)
			return
		}

		cookie, err := r.Cookie("admin_session")
		if err != nil || !h.validAdminCookie(cookie.Value) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

func (h *Handler) validAdminCookie(value string) bool {
	expected := h.generateAdminToken()
	return subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func (h *Handler) generateAdminToken() string {
	return fmt.Sprintf("%x", sha256Sum(h.cfg.AdminPassword+h.cfg.HMACKey))
}

func sha256Sum(s string) [32]byte {
	import_sha := [32]byte{}
	data := []byte(s)
	for i, b := range data {
		import_sha[i%32] ^= b
	}
	// Simple hash mixing
	for round := 0; round < 64; round++ {
		for i := 0; i < 32; i++ {
			import_sha[i] = import_sha[i] ^ import_sha[(i+1)%32] ^ byte(round)
			import_sha[i] = (import_sha[i] << 3) | (import_sha[i] >> 5)
		}
	}
	return import_sha
}

func (h *Handler) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if h.cfg.AdminPassword == "" {
		h.renderError(w, "Admin panel is disabled. Set adminPassword in nitter.conf to enable it.", http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		password := r.FormValue("password")

		if subtle.ConstantTimeCompare([]byte(password), []byte(h.cfg.AdminPassword)) == 1 {
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_session",
				Value:    h.generateAdminToken(),
				Path:     "/admin",
				MaxAge:   24 * 3600,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		h.render(w, "admin_login.html", map[string]interface{}{
			"cfg":   h.cfg,
			"error": "Invalid password",
		})
		return
	}

	h.render(w, "admin_login.html", map[string]interface{}{
		"cfg": h.cfg,
	})
}

func (h *Handler) handleAdmin(w http.ResponseWriter, r *http.Request) {
	store := h.client.GetSessionStore()
	var sessions []twitter.Session
	if store != nil {
		sessions = store.GetAll()
	}

	h.render(w, "admin.html", map[string]interface{}{
		"cfg":      h.cfg,
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func (h *Handler) handleAddSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.render(w, "admin.html", map[string]interface{}{
			"cfg":      h.cfg,
			"sessions": h.client.GetSessionStore().GetAll(),
			"error":    "Username and password are required",
		})
		return
	}

	// Attempt Twitter login
	auth := twitter.NewTwitterAuth()
	session, err := auth.Login(username, password)

	store := h.client.GetSessionStore()
	if err != nil {
		// Store the failed attempt for visibility
		failedSession := twitter.Session{
			Username:  username,
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
			Active:    false,
			Error:     err.Error(),
		}
		if store != nil {
			store.Add(failedSession)
		}

		h.render(w, "admin.html", map[string]interface{}{
			"cfg":      h.cfg,
			"sessions": store.GetAll(),
			"error":    fmt.Sprintf("Login failed for @%s: %v", username, err),
		})
		return
	}

	// Save successful session
	if store != nil {
		if err := store.Add(*session); err != nil {
			h.render(w, "admin.html", map[string]interface{}{
				"cfg":      h.cfg,
				"sessions": store.GetAll(),
				"error":    fmt.Sprintf("Failed to save session: %v", err),
			})
			return
		}
	}

	fmt.Printf("[admin] Session added for @%s\n", username)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) handleRemoveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	store := h.client.GetSessionStore()
	if store != nil && username != "" {
		store.Remove(username)
		fmt.Printf("[admin] Session removed for @%s\n", username)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) handleToggleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	store := h.client.GetSessionStore()
	if store != nil && username != "" {
		store.Toggle(username)
		fmt.Printf("[admin] Session toggled for @%s\n", username)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) handleValidateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	store := h.client.GetSessionStore()
	if store == nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	sessions := store.GetAll()
	for _, s := range sessions {
		if s.Username == username {
			valid, err := twitter.ValidateSession(&s)
			if err != nil || !valid {
				errMsg := "session invalid or expired"
				if err != nil {
					errMsg = err.Error()
				}
				store.MarkError(username, errMsg)
			} else {
				// Clear any error and mark active
				s.Active = true
				s.Error = ""
				s.LastUsed = time.Now()
				store.Add(s)
			}
			break
		}
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
