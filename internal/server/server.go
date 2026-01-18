package server

import (
	"net/http"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
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

	// Public routes
	s.router.HandleFunc("/", handlers.HandleHome(s))
	s.router.HandleFunc("/rsvp/", handlers.HandleRSVP(s))
	s.router.HandleFunc("/rsvp/submit", handlers.HandleRSVPSubmit(s))

	// Auth routes
	s.router.HandleFunc("/auth/google", s.handleGoogleLogin)
	s.router.HandleFunc("/auth/google/callback", s.handleGoogleCallback)
	s.router.HandleFunc("/auth/logout", s.handleLogout)

	// Admin routes (protected)
	s.router.HandleFunc("/admin", s.requireAuth(handlers.HandleAdminDashboard(s)))
	s.router.HandleFunc("/admin/invitations", s.requireAuth(handlers.HandleAdminInvitations(s)))
	s.router.HandleFunc("/admin/invitations/new", s.requireAuth(handlers.HandleAdminNewInvitation(s)))
	s.router.HandleFunc("/admin/invitations/create", s.requireAuth(handlers.HandleAdminCreateInvitation(s)))
	s.router.HandleFunc("/admin/invitations/edit/", s.requireAuth(handlers.HandleAdminEditInvitation(s)))
	s.router.HandleFunc("/admin/invitations/update/", s.requireAuth(handlers.HandleAdminUpdateInvitation(s)))
	s.router.HandleFunc("/admin/invitations/delete", s.requireAuth(handlers.HandleAdminDeleteInvitation(s)))
	s.router.HandleFunc("/admin/invitations/mark-sent", s.requireAuth(handlers.HandleAdminMarkSent(s)))
	s.router.HandleFunc("/admin/invitations/download-csv", s.requireAuth(handlers.HandleAdminDownloadCSV(s)))
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
