package services

import (
	"fmt"
	"laverdad-bot/db"
	googleapi "laverdad-bot/google-api"
	"laverdad-bot/locales"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

var c *cron.Cron
var chatID = int64(-4863046517)

func InitCron(botAPI *tgbotapi.BotAPI) {
	c = cron.New()

	// Creating new weekly event for club games
	_, err := c.AddFunc("0 0 * * 1", func() {
		log.Println("Creating new weekly event for club games!")

		CreateFridayEvent()
		CreateSaturdayEvent()
		CreateSundayEvent()
	})
	if err != nil {
		log.Fatal(err)
	}

	// Send Notification about Start Registration
	_, err = c.AddFunc("0 12 * * 1", func() {
		log.Println("Send Notification about Start Registration!")
		NotifyRegistrationStarted(botAPI)
	})
	if err != nil {
		log.Fatal(err)
	}

	c.Start()
}

func createNewEvent(starts_at time.Time, location string) {
	title := "Вечер клубных игр"
	description := fmt.Sprintf(`Клубные игры (фанки). 4-5 игр по спортивной мафии в дружественной атмосфере.

🗓️ %s
⏳ %s - 23:00 
📌 %s
💶 Донат на развитие клуба - 5€ с человека.

Если вы первый раз - ведущий расскажет правила и поможет влиться, во время игры будет делать небольшие комментарии 🤗
`, locales.FormatDateRU(starts_at), starts_at.Format("15:04"), location)
	event := db.Event{Title: title, Description: description, Location: location, StartsAt: starts_at}
	err := db.CreateEvent(event)
	if err != nil {
		log.Printf("Error Creating New Event: %v\n", err)
	}
	go googleapi.AddNewSheet(fmt.Sprintf("%s - %s", event.Title, event.StartsAt.Format("02.01")))
}

func nextDayOfWeek(dayOfWeek time.Weekday) time.Time {
	now := time.Now()
	daysUntilSaturday := (int(dayOfWeek) - int(now.Weekday()) + 7) % 7
	if daysUntilSaturday == 0 {
		daysUntilSaturday = 7
	}
	return now.AddDate(0, 0, daysUntilSaturday)
}

func nextDayOfWeekWithTime(dayOfWeek time.Weekday, hour int, minute int) time.Time {
	date := nextDayOfWeek(dayOfWeek)
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, date.Location())
}

func CreateFridayEvent() {
	starts_at := nextDayOfWeekWithTime(time.Friday, 18, 30)
	location := "🎙 Студия: [Calle Conejito de Málaga, 18](https://maps.app.goo.gl/K21A6KPB65FbNbcP8)"
	createNewEvent(starts_at, location)
}

func CreateSaturdayEvent() {
	starts_at := nextDayOfWeekWithTime(time.Saturday, 17, 0)
	location := "🎙 Студия: [Calle Conejito de Málaga, 18](https://maps.app.goo.gl/K21A6KPB65FbNbcP8)"
	createNewEvent(starts_at, location)
}

func CreateSundayEvent() {
	starts_at := nextDayOfWeekWithTime(time.Sunday, 18, 30)
	location := "🍕 Ресторан: [La Mafia se sienta a la mesa](https://maps.app.goo.gl/nhiYHBUkETyxuYaq9)"
	createNewEvent(starts_at, location)
}

func NotifyRegistrationStarted(botAPI *tgbotapi.BotAPI) {
	text := `Мирный привет городу, соберёмся играть в 🔴 мафию ⚫ на этой неделе?
Обратите внимание, что место и время отличаются по дням.
Для записи на игры перейдите в бот @mafia_appointment_bot`

	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := botAPI.Send(msg); err != nil {
		log.Println("notifyRegistrationStarted botAPI.Send error:", err)
	}
}
