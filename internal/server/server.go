package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/middleware"
	"github.com/AlexTLDR/evite/internal/server/handlers"
	"github.com/AlexTLDR/evite/internal/utils"
	"github.com/gorilla/sessions"
)

type Server struct {
	config       *config.Config
	db           *database.DB
	sessionStore *sessions.CookieStore
	router       *http.ServeMux
}

// GetDB implements handlers.Server interface
func (s *Server) GetDB() *database.DB {
	return s.db
}

// GetConfig implements handlers.Server interface
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// GetCurrentUser implements handlers.AdminServer interface
func (s *Server) GetCurrentUser(r *http.Request) (string, string) {
	session, err := s.sessionStore.Get(r, "auth-session")
	if err != nil {
		// Log the error but don't expose it to the user
		// Session errors can happen due to tampering or expired cookies
		return "", ""
	}
	email, _ := session.Values["email"].(string)
	name, _ := session.Values["name"].(string)
	return email, name
}

func New(cfg *config.Config, db *database.DB) *Server {
	s := &Server{
		config:       cfg,
		db:           db,
		sessionStore: sessions.NewCookieStore([]byte(cfg.SessionSecret)),
		router:       http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Static files
	fs := http.FileServer(http.Dir("./static"))
	s.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// Get trust proxy setting from config
	trustProxy := s.config.TrustProxy

	// General page rate limiting (200 requests per hour per IP)
	pageRateLimit := middleware.RateLimitByIP(200, 1*time.Hour, trustProxy)

	// Combine panic recovery with page rate limiting for public routes
	publicMiddleware := middleware.Chain(middleware.PanicRecovery, pageRateLimit)

	// Public routes with panic recovery and rate limiting
	s.router.HandleFunc("/", publicMiddleware(handlers.HandleHome(s)))
	s.router.HandleFunc("/rsvp/", publicMiddleware(handlers.HandleRSVP(s)))

	// RSVP submit with panic recovery and rate limiting
	// - 15 requests per IP per 15 minutes
	// - 15 requests per phone number per 15 minutes
	rsvpRateLimit := middleware.CombinedRateLimit(
		middleware.RateLimitByIP(15, 15*time.Minute, trustProxy),
		middleware.RateLimitByKey(15, 15*time.Minute, trustProxy, s.extractPhoneFromRequest),
	)
	rsvpMiddleware := middleware.Chain(middleware.PanicRecovery, rsvpRateLimit)
	s.router.HandleFunc("/rsvp/submit", rsvpMiddleware(handlers.HandleRSVPSubmit(s)))

	// Auth routes with panic recovery and rate limiting (10 requests per IP per 5 minutes)
	authRateLimit := middleware.RateLimitByIP(10, 5*time.Minute, trustProxy)
	authMiddleware := middleware.Chain(middleware.PanicRecovery, authRateLimit)
	s.router.HandleFunc("/auth/google", authMiddleware(s.handleGoogleLogin))
	s.router.HandleFunc("/auth/google/callback", authMiddleware(s.handleGoogleCallback))
	s.router.HandleFunc("/auth/logout", middleware.PanicRecovery(s.handleLogout))

	// Admin routes (protected) with panic recovery and rate limiting (30 requests per IP per minute)
	adminRateLimit := middleware.RateLimitByIP(30, 1*time.Minute, trustProxy)
	adminMiddleware := middleware.Chain(middleware.PanicRecovery, adminRateLimit)

	// Apply adminMiddleware consistently to all admin routes
	// Note: adminMiddleware already includes PanicRecovery, so we don't wrap it again
	s.router.HandleFunc("/admin", s.requireAuth(adminMiddleware(handlers.HandleAdminDashboard(s))))
	s.router.HandleFunc("/admin/invitations", s.requireAuth(adminMiddleware(handlers.HandleAdminInvitations(s))))
	s.router.HandleFunc("/admin/invitations/new", s.requireAuth(adminMiddleware(handlers.HandleAdminNewInvitation(s))))
	s.router.HandleFunc("/admin/invitations/create", s.requireAuth(adminMiddleware(handlers.HandleAdminCreateInvitation(s))))
	s.router.HandleFunc("/admin/invitations/edit/", s.requireAuth(adminMiddleware(handlers.HandleAdminEditInvitation(s))))
	s.router.HandleFunc("/admin/invitations/update/", s.requireAuth(adminMiddleware(handlers.HandleAdminUpdateInvitation(s))))
	s.router.HandleFunc("/admin/invitations/delete", s.requireAuth(adminMiddleware(handlers.HandleAdminDeleteInvitation(s))))
	s.router.HandleFunc("/admin/invitations/mark-sent", s.requireAuth(adminMiddleware(handlers.HandleAdminMarkSent(s))))
	s.router.HandleFunc("/admin/invitations/download-csv", s.requireAuth(adminMiddleware(handlers.HandleAdminDownloadCSV(s))))
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	return s.router
}

// Start starts the HTTP server (deprecated: use Handler() with http.Server for graceful shutdown)
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// requireAuth is a middleware that checks if user is authenticated
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionStore.Get(r, "auth-session")
		if err != nil {
			// Session error (tampered cookie, decryption failure, etc.)
			// Redirect to login to get a fresh session
			http.Redirect(w, r, "/auth/google", http.StatusSeeOther)
			return
		}

		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			http.Redirect(w, r, "/auth/google", http.StatusSeeOther)
			return
		}

		// Check if email is in whitelist
		if !s.isAdminEmail(email) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) isAdminEmail(email string) bool {
	for _, adminEmail := range s.config.AdminEmails {
		if email == adminEmail {
			return true
		}
	}
	return false
}

// extractPhoneFromRequest extracts and normalizes the phone number from the request for rate limiting
func (s *Server) extractPhoneFromRequest(r *http.Request) string {
	// Parse form if not already parsed
	if err := r.ParseForm(); err != nil {
		return ""
	}

	// Get phone from form
	phone := strings.TrimSpace(r.FormValue("phone"))
	if phone == "" {
		return ""
	}

	// Normalize phone number to prevent rate limit bypass
	// This ensures "+40721234567", "0721234567", "0721 234 567", etc. all map to the same key
	// If normalization fails, use the original phone (better than no rate limiting)
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		// If normalization fails, return original phone for rate limiting
		// This prevents bypassing rate limits with invalid phone numbers
		return phone
	}

	return normalizedPhone
}
