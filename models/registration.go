package models

import "database/sql"

func RegisterUserToEvent(db *sql.DB, userID, eventID int) error {
	_, err := db.Exec(`
        INSERT INTO registrations (user_id, event_id, status) VALUES ($1, $2, 'active')
        ON CONFLICT (user_id, event_id) DO NOTHING
    `, userID, eventID)
	return err
}
