package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int
	TelegramID int64
	ChatID     int64
	Name       string
	Nickname   string
	Phone      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func GetOrCreateUser(db *sql.DB, tgID, chatID int64) (*User, error) {
	var user User
	err := db.QueryRow(`
        SELECT id, telegram_id, chat_id, name, nickname, phone, created_at, updated_at
        FROM users WHERE telegram_id=$1
    `, tgID).Scan(&user.ID, &user.TelegramID, &user.ChatID, &user.Name, &user.Nickname, &user.Phone, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		_, err = db.Exec(`
            INSERT INTO users (telegram_id, chat_id) VALUES ($1, $2)
        `, tgID, chatID)
		if err != nil {
			return nil, err
		}
		return &User{TelegramID: tgID, ChatID: chatID}, nil
	}
	return &user, err
}

func UpdateUserProfile(db *sql.DB, tgID int64, name, nickname, phone string) error {
	_, err := db.Exec(`
        UPDATE users SET name=$1, nickname=$2, phone=$3, updated_at=now()
        WHERE telegram_id=$4
    `, name, nickname, phone, tgID)
	return err
}
