package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/middleware"
	"github.com/AlexTLDR/evite/internal/server/handlers"
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
	session, _ := s.sessionStore.Get(r, "auth-session")
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

	// General page rate limiting (200 requests per hour per IP)
	pageRateLimit := middleware.RateLimitByIP(200, 1*time.Hour)

	// Combine panic recovery with page rate limiting for public routes
	publicMiddleware := middleware.Chain(middleware.PanicRecovery, pageRateLimit)

	// Public routes with panic recovery and rate limiting
	s.router.HandleFunc("/", publicMiddleware(handlers.HandleHome(s)))
	s.router.HandleFunc("/rsvp/", publicMiddleware(handlers.HandleRSVP(s)))

	// RSVP submit with panic recovery and rate limiting
	// - 15 requests per IP per 15 minutes
	// - 15 requests per phone number per 15 minutes
	rsvpRateLimit := middleware.CombinedRateLimit(
		middleware.RateLimitByIP(15, 15*time.Minute),
		middleware.RateLimitByKey(15, 15*time.Minute, s.extractPhoneFromRequest),
	)
	rsvpMiddleware := middleware.Chain(middleware.PanicRecovery, rsvpRateLimit)
	s.router.HandleFunc("/rsvp/submit", rsvpMiddleware(handlers.HandleRSVPSubmit(s)))

	// Auth routes with panic recovery and rate limiting (10 requests per IP per 5 minutes)
	authRateLimit := middleware.RateLimitByIP(10, 5*time.Minute)
	authMiddleware := middleware.Chain(middleware.PanicRecovery, authRateLimit)
	s.router.HandleFunc("/auth/google", authMiddleware(s.handleGoogleLogin))
	s.router.HandleFunc("/auth/google/callback", authMiddleware(s.handleGoogleCallback))
	s.router.HandleFunc("/auth/logout", middleware.PanicRecovery(s.handleLogout))

	// Admin routes (protected) with panic recovery and rate limiting (30 requests per IP per minute)
	adminRateLimit := middleware.RateLimitByIP(30, 1*time.Minute)
	adminMiddleware := middleware.Chain(middleware.PanicRecovery, adminRateLimit)

	s.router.HandleFunc("/admin", middleware.PanicRecovery(s.requireAuth(handlers.HandleAdminDashboard(s))))
	s.router.HandleFunc("/admin/invitations", middleware.PanicRecovery(s.requireAuth(handlers.HandleAdminInvitations(s))))
	s.router.HandleFunc("/admin/invitations/new", middleware.PanicRecovery(s.requireAuth(handlers.HandleAdminNewInvitation(s))))
	s.router.HandleFunc("/admin/invitations/create", middleware.PanicRecovery(s.requireAuth(adminMiddleware(handlers.HandleAdminCreateInvitation(s)))))
	s.router.HandleFunc("/admin/invitations/edit/", middleware.PanicRecovery(s.requireAuth(handlers.HandleAdminEditInvitation(s))))
	s.router.HandleFunc("/admin/invitations/update/", middleware.PanicRecovery(s.requireAuth(adminMiddleware(handlers.HandleAdminUpdateInvitation(s)))))
	s.router.HandleFunc("/admin/invitations/delete", middleware.PanicRecovery(s.requireAuth(adminMiddleware(handlers.HandleAdminDeleteInvitation(s)))))
	s.router.HandleFunc("/admin/invitations/mark-sent", middleware.PanicRecovery(s.requireAuth(adminMiddleware(handlers.HandleAdminMarkSent(s)))))
	s.router.HandleFunc("/admin/invitations/download-csv", middleware.PanicRecovery(s.requireAuth(handlers.HandleAdminDownloadCSV(s))))
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// requireAuth is a middleware that checks if user is authenticated
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.sessionStore.Get(r, "auth-session")

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

// extractPhoneFromRequest extracts the phone number from the request for rate limiting
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

	// Return the phone as-is for rate limiting key
	// We don't normalize here to avoid expensive operations in middleware
	return phone
}
