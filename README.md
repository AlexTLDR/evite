# Evite - Event Invitation System

A simple, elegant RSVP system for events and parties with magic links and admin management.

## Features

- 🔗 **Magic Links** - No login required for guests
- 📱 **WhatsApp Integration** - Easy copy-paste invite messages
- 👥 **Guest Management** - Track invitations, opens, and responses
- 🔒 **Google OAuth** - Secure admin access with email whitelist
- 📊 **Dashboard** - View attendance statistics and guest responses
- 🌍 **Bilingual** - Romanian and English support
- 📝 **Response History** - Track changes with deadline enforcement
- 🏷️ **Name Tags** - Collect preferred names for table seating

## Use Cases

Perfect for:
- Baptisms
- Weddings
- Birthday parties
- Corporate events
- Any event requiring RSVP tracking

## Tech Stack

- **Backend**: Go 1.25+
- **Database**: PostgreSQL with Goose migrations
- **Templates**: Templ
- **Auth**: Google OAuth 2.0
- **Sessions**: Gorilla Sessions

## Setup

### Prerequisites

- Go 1.25 or higher
- Google OAuth credentials

### Installation

1. Clone the repository:
```bash
git clone https://github.com/AlexTLDR/evite.git
cd evite
```

2. Install dependencies:
```bash
go mod download
```

3. Copy the example environment file:
```bash
cp .env.example .env
```

4. Configure your `.env` file with:
   - Google OAuth credentials
   - Admin email addresses
   - Event details (date, church, restaurant)
   - RSVP deadline

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:8080/auth/google/callback` (or your domain)
6. Copy Client ID and Client Secret to `.env`

### Running the Application

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080` (or the port specified in `.env`)

## Database Migrations

Migrations are managed with [Goose](https://github.com/pressly/goose).

### Run migrations:
```bash
go run cmd/server/main.go
```
Migrations run automatically on startup.

### Create a new migration:
```bash
goose -dir migrations create migration_name sql
```

### Manual migration commands:
```bash
# Up
goose -dir migrations sqlite3 ./evite.db up

# Down
goose -dir migrations sqlite3 ./evite.db down

# Status
goose -dir migrations sqlite3 ./evite.db status
```

## Usage

### Admin Workflow

1. Login with Google (whitelisted email)
2. Create new invitation with guest name and phone
3. Copy the generated WhatsApp message
4. Send via WhatsApp manually
5. Mark invitation as sent
6. Track opens and responses in dashboard

### Guest Workflow

1. Receive WhatsApp message with magic link
2. Click link to open RSVP form
3. Fill in attendance details:
   - Attending yes/no
   - Plus one (with name for table tag)
   - Number of kids
   - Preferred name for table tag
   - Optional comments
4. Submit response
5. Can edit until deadline

## Project Structure

```
evite/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database models and queries
│   ├── i18n/            # Internationalization
│   └── server/          # HTTP server and handlers
├── migrations/          # Database migrations
├── static/              # Static assets (CSS, JS)
├── templates/           # Templ templates
├── .env.example         # Example environment variables
└── go.mod               # Go module definition
```

## Environment Variables

See `.env.example` for all available configuration options.

## License

See LICENSE file for details.

