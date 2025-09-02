package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
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

type Event struct {
	ID          int
	Title       string
	Description string
	Location    string
	StartsAt    time.Time
}

type Registration struct {
	ID        int
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	StartsAt  time.Time
}

type AdminRegistration struct {
	ID         int
	Title      string
	Name       string
	Nickname   string
	TelegramID int64
}

type RegistrationLine struct {
	ID           int
	TelegramLink string
	UserName     sql.NullString
	Name         string
	NickName     string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var DB *sql.DB

func InitDB(connStr string) {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Таблицы
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		chat_id BIGINT NOT NULL,
		name TEXT,
		nickname TEXT
	);

	CREATE TABLE IF NOT EXISTS events (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		starts_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS registrations (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		UNIQUE (user_id, event_id)
	);
	`)
	if err != nil {
		log.Fatal(err)
	}
}

// Проверка, есть ли пользователь
func UserExists(telegramID int64) bool {
	var exists bool
	err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id=$1)`, telegramID).Scan(&exists)
	if err != nil {
		log.Println("UserExists error:", err)
		return false
	}
	return exists
}

func GetUser(telegramID int64) User {
	var user User
	q := `
        SELECT id, telegram_id, chat_id, name, nickname, phone, created_at, updated_at
        FROM users WHERE telegram_id=$1
    `
	DB.QueryRow(q, telegramID).Scan(&user.ID, &user.TelegramID, &user.ChatID, &user.Name, &user.Nickname, &user.Phone, &user.CreatedAt, &user.UpdatedAt)

	return user
}

func GetOrCreateUser(telegramID, chatID int64, userName string) (*User, error) {
	var user User
	err := DB.QueryRow(`
        SELECT id, telegram_id, chat_id, name, nickname, phone, created_at, updated_at
        FROM users WHERE telegram_id=$1
    `, telegramID).Scan(&user.ID, &user.TelegramID, &user.ChatID, &user.Name, &user.Nickname, &user.Phone, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		_, err = DB.Exec(`
            INSERT INTO users (telegram_id, chat_id, username) VALUES ($1, $2, $3)
        `, telegramID, chatID, userName)
		if err != nil {
			return nil, err
		}
		return &User{TelegramID: telegramID, ChatID: chatID}, nil
	}
	return &user, err
}

// Создание пользователя
func CreateUser(telegramID int64, chatID int64) error {
	_, err := DB.Exec(`INSERT INTO users (telegram_id, chat_id) VALUES ($1, $2) ON CONFLICT (telegram_id) DO NOTHING`, telegramID, chatID)
	return err
}

// Обновление имени
func UpdateUserName(telegramID int64, name string) error {
	_, err := DB.Exec(`UPDATE users SET name=$1 WHERE telegram_id=$2`, name, telegramID)
	return err
}

// Обновление ника
func UpdateUserNickname(telegramID int64, nickname string) error {
	_, err := DB.Exec(`UPDATE users SET nickname=$1 WHERE telegram_id=$2`, nickname, telegramID)
	return err
}

// Получить список событий
func GetEvents() []Event {
	rows, err := DB.Query(`SELECT id, title, description, location, starts_at FROM events WHERE starts_at >= CURRENT_DATE ORDER BY starts_at`)
	if err != nil {
		log.Println("GetEvents error:", err)
		return nil
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &e.StartsAt)
		if err != nil {
			log.Println("GetEvents scan error:", err)
			continue
		}
		events = append(events, e)
	}
	return events
}

func CreateEvent(event Event) error {
	_, err := DB.Exec(`INSERT INTO events (title, description, location, starts_at) VALUES ($1, $2, $3, $4)`, event.Title, event.Description, event.Location, event.StartsAt)
	return err
}

func RegistrationExists(eventID int64, userID int64) bool {
	var exists bool
	err := DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM registrations WHERE event_id=$1 AND user_id=$2)`, eventID, userID).Scan(&exists)
	if err != nil {
		log.Println("UserExists error:", err)
		return false
	}
	return exists
}

func GetRegistrationsByEvent(eventID int) []AdminRegistration {
	var regs []AdminRegistration
	q := `
	SELECT r.id, e.title, u.name, u.nickname, u.telegram_id
	FROM registrations r
	JOIN events e ON r.event_id = e.id
	JOIN users u ON r.user_id = u.id
	WHERE e.id = $1
	ORDER BY r.created_at
	`
	rows, err := DB.Query(q, eventID)
	if err != nil {
		log.Printf("GetRegistrations error: %v\n", err)
		return regs
	}
	defer rows.Close()

	for rows.Next() {
		var r AdminRegistration

		err := rows.Scan(&r.ID, &r.Title, &r.Name, &r.Nickname, &r.TelegramID)
		if err != nil {
			log.Printf("GetRegistrations scan error: %v\n", err)
			continue
		}
		regs = append(regs, r)
	}

	return regs
}

func GetEventParticipantsCount(eventID int) (int, error) {
	var count int

	err := DB.QueryRow(`SELECT COUNT(*) FROM registrations WHERE registrations.event_id = $1`, eventID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetEventParticipantsCount scan error: %v", err)
	}

	return count, nil
}

// Зарегистрировать пользователя на событие
func RegisterUserToEvent(telegramID int64, eventID int) error {
	var userID int
	err := DB.QueryRow(`SELECT id FROM users WHERE telegram_id=$1`, telegramID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("пользователь не найден")
	}

	_, err = DB.Exec(`INSERT INTO registrations (user_id, event_id) VALUES ($1, $2)`, userID, eventID)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрироваться: %v", err)
	}
	return nil
}

func GetRegistrationLine(telegramID int, eventID int) (RegistrationLine, error) {
	var line RegistrationLine
	q := `
		SELECT r.id, u.username, u.name, u.nickname, r.status, r.created_at, r.updated_at
		FROM registrations r
		JOIN users u ON u.id = r.user_id
		WHERE r.event_id = $1
		  AND u.telegram_id = $2
		LIMIT 1
	`
	err := DB.QueryRow(q, eventID, telegramID).Scan(&line.ID, &line.UserName, &line.Name, &line.NickName, &line.Status, &line.CreatedAt, &line.UpdatedAt)
	if err != nil {
		log.Printf("GetRegistrationLine QueryRow ERROR! %v\n", err)
		return line, fmt.Errorf("RegistrationLine query error: %v", err)
	}
	line.TelegramLink = fmt.Sprintf("tg://user?id=%s", strconv.Itoa(telegramID))

	return line, nil
}

// Получить регистрации пользователя
func GetUserRegistrations(telegramID int64) []Registration {
	var regs []Registration

	rows, err := DB.Query(`
	SELECT e.id, e.title, e.starts_at
	FROM registrations r
	JOIN users u ON r.user_id = u.id
	JOIN events e ON r.event_id = e.id
	WHERE u.telegram_id=$1
	ORDER BY e.starts_at`, telegramID)
	if err != nil {
		log.Printf("GetUserRegistrations error: %v\n", err)
		return regs
	}
	defer rows.Close()

	for rows.Next() {
		var r Registration
		err := rows.Scan(&r.ID, &r.Title, &r.StartsAt)
		if err != nil {
			log.Printf("GetUserRegistrations scan error: %v\n", err)
			continue
		}
		regs = append(regs, r)
	}

	return regs
}

func GetRegistrationByID(regID int) Registration {
	var reg Registration
	err := DB.QueryRow(`SELECT r.id, r.created_at, r.updated_at
	FROM registrations r
	WHERE r.id = $1`, regID).Scan(&reg.ID, &reg.CreatedAt, &reg.UpdatedAt)
	if err != nil {
		fmt.Printf("ERROR SELECT registrations with id=%d! Error: %v\n", regID, err)
	}
	return reg
}

func GetRegistrationID(tgID int64, eventID int) int {
	var regID int
	err := DB.QueryRow(`SELECT r.id FROM registrations r JOIN users u ON u.id = r.user_id WHERE u.telegram_id=$1 AND r.event_id=$2`, tgID, eventID).Scan(&regID)
	if err != nil {
		fmt.Printf("ERROR SELECT registrations for user.telegram_id=%d and event_id=%d! Error: %v\n", int(tgID), eventID, err)
	}
	return regID
}

// Отмена регистрации
func CancelUserRegistration(telegramID int64, eventID int) error {
	var userID int
	err := DB.QueryRow(`SELECT id FROM users WHERE telegram_id=$1`, telegramID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("ERROR SELECT user with telegram_id: %d", int(telegramID))
	}

	_, err = DB.Exec(`DELETE FROM registrations WHERE user_id=$1 AND event_id=$2`, userID, eventID)
	if err != nil {
		return fmt.Errorf("не удалось отменить регистрацию: %v", err)
	}
	return nil
}

func UpdateRegistrationNotificationStatus(eventIDS []string, column string) error {
	var regIDS []string
	rows, err := DB.Query(`SELECT id FROM registrations WHERE event_id IN($1)`, strings.Join(eventIDS, ", "))
	if err != nil {
		return fmt.Errorf("UpdateRegistrationNotificationStatus error: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var regID int
		err = rows.Scan(&regID)
		if err != nil {
			log.Printf("UpdateRegistrationNotificationStatus scan error: %v\n", err)
			continue
		}
		regIDS = append(regIDS, strconv.Itoa(regID))
	}

	q := fmt.Sprintf("UPDATE registrations SET %s='t' WHERE id IN(%s)", column, strings.Join(regIDS, ", "))
	_, err = DB.Exec(q)
	if err != nil {
		return fmt.Errorf("ERROR UPDATE registrations: %v", err)
	}
	return nil
}

func FetchEvent(id int64) (Event, error) {
	var out Event
	q := `select id, title, coalesce(description,''), location, starts_at from events where id = $1 limit 1`
	err := DB.QueryRow(q, id).Scan(&out.ID, &out.Title, &out.Description, &out.Location, &out.StartsAt)

	return out, err
}

func GetUpcomingEvents(d time.Duration) []Event {
	rows, err := DB.Query(`
        SELECT id, title, description, location, starts_at
        FROM events
        WHERE starts_at > NOW() AND starts_at <= NOW() + $1::interval
    `, fmt.Sprintf("%f hour", d.Hours()))
	if err != nil {
		log.Println("ERROR Query GetUpcomingEvents:", err)
		return nil
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Location, &e.StartsAt)
		if err != nil {
			log.Println("ERROR Scan GetUpcomingEvents:", err)
			continue
		}
		events = append(events, e)
	}
	return events
}

func GetEventParticipants(eventID int) ([]User, error) {
	q := fmt.Sprintf(`
	SELECT u.id, u.
	FROM registrations r
	JOIN users u ON r.user_id = u.id
	WHERE r.event_id = %d
	`, eventID)
	rows, err := DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.ChatID, &u.TelegramID, &u.Name, &u.Nickname, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			log.Println("GetEventParticipants scan error:", err)
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func GetEventParticipantsWithFlag(eventID int, column string) []User {
	q := fmt.Sprintf(`
	SELECT u.chat_id
	FROM registrations r
	JOIN users u ON r.user_id = u.id
	WHERE r.event_id = %d
	AND r.%s = 'f'
	`, eventID, column)
	rows, err := DB.Query(q)
	if err != nil {
		log.Println("GetEventParticipants error:", err)
		return nil
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ChatID)
		if err != nil {
			log.Println("GetEventParticipants scan error:", err)
			continue
		}
		users = append(users, u)
	}
	return users
}
