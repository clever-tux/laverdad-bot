package models

import (
	"database/sql"
	"time"
)

type Event struct {
	ID          int
	Title       string
	Description string
	StartsAt    time.Time
}

func GetAllEvents(db *sql.DB) ([]Event, error) {
	rows, err := db.Query(`SELECT id, title, description, starts_at FROM events ORDER BY starts_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.StartsAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func GetEventByID(db *sql.DB, id int) (*Event, error) {
	var e Event
	err := db.QueryRow(`SELECT id, title, description, starts_at FROM events WHERE id=$1`, id).
		Scan(&e.ID, &e.Title, &e.Description, &e.StartsAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
