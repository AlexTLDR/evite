package database

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateResponse creates a new response and marks previous responses as not latest
func (db *DB) CreateResponse(invitationID int64, attending, plusOne bool, plusOneName, guestNameTag string, kidsCount int, menuPreference, companionMenuPreference, comment string) (*Response, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark all previous responses as not latest
	_, err = tx.Exec(
		`UPDATE responses SET is_latest = FALSE WHERE invitation_id = $1`,
		invitationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update previous responses: %w", err)
	}

	// Insert new response
	var plusOneNameSQL, menuPreferenceSQL, companionMenuPreferenceSQL, commentSQL interface{}
	if plusOneName != "" {
		plusOneNameSQL = plusOneName
	}
	if menuPreference != "" {
		menuPreferenceSQL = menuPreference
	}
	if companionMenuPreference != "" {
		companionMenuPreferenceSQL = companionMenuPreference
	}
	if comment != "" {
		commentSQL = comment
	}

	var id int64
	err = tx.QueryRow(
		`INSERT INTO responses (invitation_id, attending, plus_one, plus_one_name, guest_name_tag, kids_count, menu_preference, companion_menu_preference, comment, is_latest)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE) RETURNING id`,
		invitationID, attending, plusOne, plusOneNameSQL, guestNameTag, kidsCount, menuPreferenceSQL, companionMenuPreferenceSQL, commentSQL,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	// Update invitation responded_at
	_, err = tx.Exec(
		`UPDATE invitations SET responded_at = $1 WHERE id = $2`,
		time.Now(), invitationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update invitation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return db.GetResponseByID(id)
}

// GetResponseByID retrieves a response by ID
func (db *DB) GetResponseByID(id int64) (*Response, error) {
	resp := &Response{}
	err := db.QueryRow(
		`SELECT id, invitation_id, attending, plus_one, plus_one_name, guest_name_tag, kids_count, menu_preference, companion_menu_preference, comment, submitted_at, is_latest
		 FROM responses WHERE id = $1`,
		id,
	).Scan(&resp.ID, &resp.InvitationID, &resp.Attending, &resp.PlusOne, &resp.PlusOneName,
		&resp.GuestNameTag, &resp.KidsCount, &resp.MenuPreference, &resp.CompanionMenuPreference, &resp.Comment, &resp.SubmittedAt, &resp.IsLatest)

	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	return resp, nil
}

// GetLatestResponseByInvitationID retrieves the latest response for an invitation
func (db *DB) GetLatestResponseByInvitationID(invitationID int64) (*Response, error) {
	resp := &Response{}
	err := db.QueryRow(
		`SELECT id, invitation_id, attending, plus_one, plus_one_name, guest_name_tag, kids_count, menu_preference, companion_menu_preference, comment, submitted_at, is_latest
		 FROM responses WHERE invitation_id = $1 AND is_latest = TRUE`,
		invitationID,
	).Scan(&resp.ID, &resp.InvitationID, &resp.Attending, &resp.PlusOne, &resp.PlusOneName,
		&resp.GuestNameTag, &resp.KidsCount, &resp.MenuPreference, &resp.CompanionMenuPreference, &resp.Comment, &resp.SubmittedAt, &resp.IsLatest)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest response: %w", err)
	}

	return resp, nil
}

// GetAllResponsesByInvitationID retrieves all responses for an invitation (history)
func (db *DB) GetAllResponsesByInvitationID(invitationID int64) ([]*Response, error) {
	rows, err := db.Query(
		`SELECT id, invitation_id, attending, plus_one, plus_one_name, guest_name_tag, kids_count, menu_preference, companion_menu_preference, comment, submitted_at, is_latest
		 FROM responses WHERE invitation_id = $1 ORDER BY submitted_at DESC`,
		invitationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get responses: %w", err)
	}
	defer rows.Close()

	var responses []*Response
	for rows.Next() {
		resp := &Response{}
		err := rows.Scan(&resp.ID, &resp.InvitationID, &resp.Attending, &resp.PlusOne, &resp.PlusOneName,
			&resp.GuestNameTag, &resp.KidsCount, &resp.MenuPreference, &resp.CompanionMenuPreference, &resp.Comment, &resp.SubmittedAt, &resp.IsLatest)
		if err != nil {
			return nil, fmt.Errorf("failed to scan response: %w", err)
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// GetAllInvitationsWithResponses retrieves all invitations with their latest responses
func (db *DB) GetAllInvitationsWithResponses() ([]*InvitationWithResponse, error) {
	rows, err := db.Query(
		`SELECT
			i.id, i.guest_name, i.phone, i.token, i.invite_message, i.sent_at, i.opened_at, i.responded_at, i.created_at,
			r.id, r.invitation_id, r.attending, r.plus_one, r.plus_one_name, r.guest_name_tag, r.kids_count, r.menu_preference, r.companion_menu_preference, r.comment, r.submitted_at, r.is_latest
		 FROM invitations i
		 LEFT JOIN responses r ON i.id = r.invitation_id AND r.is_latest = TRUE
		 ORDER BY i.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations with responses: %w", err)
	}
	defer rows.Close()

	var results []*InvitationWithResponse
	for rows.Next() {
		iwr := &InvitationWithResponse{}
		var respID sql.NullInt64
		var respInvID sql.NullInt64
		var respAttending sql.NullBool
		var respPlusOne sql.NullBool
		var respPlusOneName sql.NullString
		var respGuestNameTag sql.NullString
		var respKidsCount sql.NullInt64
		var respMenuPreference sql.NullString
		var respCompanionMenuPreference sql.NullString
		var respComment sql.NullString
		var respSubmittedAt sql.NullTime
		var respIsLatest sql.NullBool

		err := rows.Scan(
			&iwr.ID, &iwr.GuestName, &iwr.Phone, &iwr.Token, &iwr.InviteMessage,
			&iwr.SentAt, &iwr.OpenedAt, &iwr.RespondedAt, &iwr.CreatedAt,
			&respID, &respInvID, &respAttending, &respPlusOne, &respPlusOneName,
			&respGuestNameTag, &respKidsCount, &respMenuPreference, &respCompanionMenuPreference, &respComment, &respSubmittedAt, &respIsLatest,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation with response: %w", err)
		}

		if respID.Valid {
			iwr.Response = &Response{
				ID:                      respID.Int64,
				InvitationID:            respInvID.Int64,
				Attending:               respAttending.Bool,
				PlusOne:                 respPlusOne.Bool,
				PlusOneName:             respPlusOneName,
				GuestNameTag:            respGuestNameTag.String,
				KidsCount:               int(respKidsCount.Int64),
				MenuPreference:          respMenuPreference,
				CompanionMenuPreference: respCompanionMenuPreference,
				Comment:                 respComment,
				SubmittedAt:             respSubmittedAt.Time,
				IsLatest:                respIsLatest.Bool,
			}
		}

		results = append(results, iwr)
	}

	return results, nil
}
