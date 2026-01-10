package server

import (
	"fmt"
	"net/http"
)

// Public handlers
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Baptism Invitation</title>
		</head>
		<body>
			<h1>Welcome to the Baptism Invitation System</h1>
			<p><a href="/auth/google">Admin Login</a></p>
		</body>
		</html>
	`)
}

func (s *Server) handleRSVP(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement RSVP form
	w.Write([]byte("RSVP Form - Coming soon"))
}

func (s *Server) handleRSVPSubmit(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement RSVP submission
	w.Write([]byte("RSVP Submit - Coming soon"))
}

// Admin handlers
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement admin dashboard
	email, name := s.getCurrentUser(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Admin Dashboard</title>
		</head>
		<body>
			<h1>Admin Dashboard</h1>
			<p>Welcome, %s (%s)</p>
			<nav>
				<a href="/admin/invitations">Invitations</a> |
				<a href="/auth/logout">Logout</a>
			</nav>
		</body>
		</html>
	`, name, email)
}

func (s *Server) handleAdminInvitations(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement invitations list
	w.Write([]byte("Admin Invitations List - Coming soon"))
}

func (s *Server) handleAdminNewInvitation(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement new invitation form
	w.Write([]byte("New Invitation Form - Coming soon"))
}

func (s *Server) handleAdminCreateInvitation(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement create invitation
	w.Write([]byte("Create Invitation - Coming soon"))
}

func (s *Server) handleAdminMarkSent(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement mark as sent
	w.Write([]byte("Mark as Sent - Coming soon"))
}

