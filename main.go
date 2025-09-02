package main

import (
	"laverdad-bot/bot"
	"laverdad-bot/db"
	googleapi "laverdad-bot/google-api"
	"laverdad-bot/services"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN не задан")
	}

	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	botAPI.Debug = true

	log.Printf("Бот запущен: %s", botAPI.Self.UserName)

	db.InitDB(os.Getenv("DATABASE_URL"))

	googleapi.InitSheetService()

	// Запуск горутины уведомлений
	go bot.StartNotifications(botAPI)

	services.InitCron(botAPI)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := botAPI.GetUpdatesChan(updateConfig)

	for update := range updates {
		go bot.HandleUpdate(botAPI, update)
	}
}
