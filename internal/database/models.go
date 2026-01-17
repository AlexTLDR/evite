package database

import (
	"database/sql"
	"time"
)

type Invitation struct {
	ID            int64
	GuestName     string
	Phone         string
	Token         string
	InviteMessage string
	SentAt        sql.NullTime
	OpenedAt      sql.NullTime
	RespondedAt   sql.NullTime
	CreatedAt     time.Time
}

type Response struct {
	ID                      int64
	InvitationID            int64
	Attending               bool
	PlusOne                 bool
	PlusOneName             sql.NullString
	GuestNameTag            string
	KidsCount               int
	MenuPreference          sql.NullString
	CompanionMenuPreference sql.NullString
	Comment                 sql.NullString
	SubmittedAt             time.Time
	IsLatest                bool
}

type InvitationWithResponse struct {
	Invitation
	Response *Response
}
