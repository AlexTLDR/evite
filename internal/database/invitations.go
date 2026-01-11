package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func GenerateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateInvitation creates a new invitation with a unique token
func (db *DB) CreateInvitation(guestName, phone, inviteMessage string) (*Invitation, error) {
	// Generate unique token with retry logic
	var token string
	var err error
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		token, err = GenerateToken()
		if err != nil {
			return nil, err
		}

		// Check if token already exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM invitations WHERE token = ?)", token).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("failed to check token uniqueness: %w", err)
		}

		if !exists {
			break
		}

		if i == maxRetries-1 {
			return nil, fmt.Errorf("failed to generate unique token after %d retries", maxRetries)
		}
	}

	result, err := db.Exec(
		`INSERT INTO invitations (guest_name, phone, token, invite_message) 
		 VALUES (?, ?, ?, ?)`,
		guestName, phone, token, inviteMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return db.GetInvitationByID(id)
}

// GetInvitationByID retrieves an invitation by ID
func (db *DB) GetInvitationByID(id int64) (*Invitation, error) {
	inv := &Invitation{}
	err := db.QueryRow(
		`SELECT id, guest_name, phone, token, invite_message, sent_at, opened_at, responded_at, created_at
		 FROM invitations WHERE id = ?`,
		id,
	).Scan(&inv.ID, &inv.GuestName, &inv.Phone, &inv.Token, &inv.InviteMessage,
		&inv.SentAt, &inv.OpenedAt, &inv.RespondedAt, &inv.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return inv, nil
}

// GetInvitationByToken retrieves an invitation by token
func (db *DB) GetInvitationByToken(token string) (*Invitation, error) {
	inv := &Invitation{}
	err := db.QueryRow(
		`SELECT id, guest_name, phone, token, invite_message, sent_at, opened_at, responded_at, created_at
		 FROM invitations WHERE token = ?`,
		token,
	).Scan(&inv.ID, &inv.GuestName, &inv.Phone, &inv.Token, &inv.InviteMessage,
		&inv.SentAt, &inv.OpenedAt, &inv.RespondedAt, &inv.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return inv, nil
}

// GetAllInvitations retrieves all invitations
func (db *DB) GetAllInvitations() ([]*Invitation, error) {
	rows, err := db.Query(
		`SELECT id, guest_name, phone, token, invite_message, sent_at, opened_at, responded_at, created_at
		 FROM invitations ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}
	defer rows.Close()

	var invitations []*Invitation
	for rows.Next() {
		inv := &Invitation{}
		err := rows.Scan(&inv.ID, &inv.GuestName, &inv.Phone, &inv.Token, &inv.InviteMessage,
			&inv.SentAt, &inv.OpenedAt, &inv.RespondedAt, &inv.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}

	return invitations, nil
}

// MarkAsSent marks an invitation as sent
func (db *DB) MarkAsSent(id int64) error {
	_, err := db.Exec(
		`UPDATE invitations SET sent_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark invitation as sent: %w", err)
	}
	return nil
}

// MarkAsOpened marks an invitation as opened (when guest visits RSVP page)
func (db *DB) MarkAsOpened(id int64) error {
	_, err := db.Exec(
		`UPDATE invitations SET opened_at = ? WHERE id = ? AND opened_at IS NULL`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark invitation as opened: %w", err)
	}
	return nil
}

// UpdateInvitation updates an invitation's guest name and phone
func (db *DB) UpdateInvitation(id int64, guestName, phone string) error {
	_, err := db.Exec(
		`UPDATE invitations SET guest_name = ?, phone = ? WHERE id = ?`,
		guestName, phone, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}
	return nil
}

// DeleteInvitation deletes an invitation and all its responses
func (db *DB) DeleteInvitation(id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all responses first
	_, err = tx.Exec(`DELETE FROM responses WHERE invitation_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete responses: %w", err)
	}

	// Delete the invitation
	_, err = tx.Exec(`DELETE FROM invitations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
