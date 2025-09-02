package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"laverdad-bot/db"
	googleapi "laverdad-bot/google-api"
	"laverdad-bot/locales"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type State string

const (
	StateNone          State = ""
	StateEnterName     State = "enter_name"
	StateEnterNickname State = "enter_nickname"
	StateRegister      State = "register"
)

var userStates = map[int64]State{}

var laVerdadChatID = -4863046517

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message != nil {
		handleMessage(bot, update.Message)
	} else if update.CallbackQuery != nil {
		handleCallback(bot, update.CallbackQuery)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	tgID := msg.From.ID
	tgUser := msg.From

	// Проверяем состояние пользователя
	state := userStates[chatID]

	switch state {
	case StateEnterName:
		db.UpdateUserName(int64(tgID), msg.Text)
		userStates[chatID] = StateEnterNickname
		sendText(bot, chatID, "Отлично! Теперь введи свой игровой *ник*:")
		return

	case StateEnterNickname:
		db.UpdateUserNickname(int64(tgID), msg.Text)
		userStates[chatID] = StateNone
		sendText(bot, chatID, "Готово! Теперь можешь использовать команды:\n/events — Список событий\n/my — Мои регистрации")
		return
	}

	// Команды
	switch msg.Text {
	case "/start":
		text := "Привет! Я бот клуба спортивной мафии *La Verdad*. \nС помощью меня можно записаться на игры и не только 😉"
		sendText(bot, chatID, text)

		// Если пользователь новый — добавляем и спрашиваем имя

		user, err := db.GetOrCreateUser(int64(tgID), chatID, tgUser.UserName)
		if err != nil {
			log.Println("db.GetOrCreateUser error:", err)
		}
		if (user.Name == "" || user.Nickname == "") && state == StateNone {
			userStates[chatID] = StateEnterName
			sendText(bot, chatID, "Пожалуйста пройди небольшую регистрацию\n\nВведи своё *имя*:")
			return
		}

	case "/events":
		events := db.GetEvents()
		if len(events) == 0 {
			sendText(bot, chatID, "Пока нет доступных событий.")
			return
		}

		text := "Доступные мероприятия:\n\n"
		markup := tgbotapi.NewInlineKeyboardMarkup()
		var rows [][]tgbotapi.InlineKeyboardButton
		for _, e := range events {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%s — %s", e.Title, e.StartsAt.Format("02.01 15:04")),
					fmt.Sprintf("ev_%d", e.ID),
				),
			)
			rows = append(rows, row)
		}
		markup.InlineKeyboard = rows
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = markup
		bot.Send(msg)

	case "/my":
		registrations := db.GetUserRegistrations(int64(tgID))
		if len(registrations) == 0 {
			sendText(bot, chatID, "Ты пока никуда не записан.")
			return
		}
		for _, r := range registrations {
			text := fmt.Sprintf("*%s*\nСтарт: %s", r.Title, r.StartsAt.Format("02.01.2006 15:04"))
			btn := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Отменить", fmt.Sprintf("cancel_%d", r.ID)),
				),
			)
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = btn
			bot.Send(msg)
		}

	default:
		// проверяем админа
		if IsAdmin(tgID) {
			HandleAdmin(bot, msg)
			return
		}
		sendText(bot, chatID, "Неизвестная команда. Доступные команды:\n/events — список событий\n/my — мои регистрации")
	}
}

func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	tgID := callback.From.ID
	mesgID := callback.Message.MessageID

	if strings.HasPrefix(data, "ev_") {
		eventIDStr := strings.TrimPrefix(data, "ev_")
		eventID, _ := strconv.Atoi(eventIDStr)

		ev, err := db.FetchEvent(int64(eventID))
		if err != nil {
			log.Println("fetchEvent:", err)
			sendText(bot, chatID, "Ошибка: Не удалось загрузить событие.")
			return
		}

		// Already registered?
		user := db.GetUser(tgID)
		regExists := db.RegistrationExists(int64(eventID), int64(user.ID))

		text := ev.Description

		if regExists {
			text += "\n\n✅ *Вы уже зарегистрированы*"
		}

		edit := tgbotapi.NewEditMessageText(chatID, mesgID, text)
		edit.ParseMode = "Markdown"

		// Кнопка для начала регистрации
		if !regExists {
			btn := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📝 Записаться", fmt.Sprintf("register_%d", eventID)),
				),
			)
			edit.ReplyMarkup = &btn
		}

		if _, err := bot.Send(edit); err != nil {
			log.Println("Event list event:", err)
		}

		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return

	} else if strings.HasPrefix(data, "admin_ev_") {
		eventIDStr := strings.TrimPrefix(data, "admin_ev_")
		eventID, _ := strconv.Atoi(eventIDStr)

		event, err := db.FetchEvent(int64(eventID))
		if err != nil {
			sendText(bot, chatID, "Ошибка: Не удалось загрузить событие.")
			return
		}

		text := fmt.Sprintf("%s — %s\n", event.Title, event.StartsAt.Format("02.01 15:04"))

		regs := db.GetRegistrationsByEvent(eventID)
		if len(regs) == 0 {
			text += "Нет регистраций на мероприятие!"
		} else {
			for _, r := range regs {
				text += fmt.Sprintf("- [%s  (%s)](tg://user?id=%s)\n", r.Name, r.Nickname, strconv.Itoa(int(r.TelegramID)))
			}
		}

		sendText(bot, chatID, text)

		return

	} else if strings.HasPrefix(data, "register_") {
		eventIDStr := strings.TrimPrefix(data, "register_")
		eventID, _ := strconv.Atoi(eventIDStr)

		err := db.RegisterUserToEvent(int64(tgID), eventID)
		if err != nil {
			sendText(bot, chatID, fmt.Sprintf("RegisterUserToEvent Error: %v", err))
			return
		}

		event, err := db.FetchEvent(int64(eventID))
		if err == nil {
			sheetName := fmt.Sprintf("%s - %s", event.Title, event.StartsAt.Format("02.01"))
			line, _ := db.GetRegistrationLine(int(tgID), eventID)

			go googleapi.AddRegistrationToSheet(sheetName, line)
			sendText(bot, chatID, "✅ Ты успешно зарегистрирован на событие!")
		} else {
			log.Printf("Error FetchEvent with id=%d\n", int(eventID))
		}

	} else if strings.HasPrefix(data, "cancel_") {
		eventIDStr := strings.TrimPrefix(data, "cancel_")
		eventID, _ := strconv.Atoi(eventIDStr)

		regID := db.GetRegistrationID(int64(tgID), eventID)
		event, _ := db.FetchEvent(int64(eventID))
		sheetName := fmt.Sprintf("%s - %s", event.Title, event.StartsAt.Format("02.01"))
		go googleapi.UpdateRegistrationStateToSheet(regID, sheetName, time.Now())

		err := db.CancelUserRegistration(int64(tgID), eventID)
		if err != nil {
			sendText(bot, chatID, fmt.Sprintf("Ошибка: %v", err))
			return
		}

		sendText(bot, chatID, "❌ Регистрация отменена.")
	}

	cb := tgbotapi.NewCallback(callback.ID, "Регистрация успешна!")
	bot.Send(cb)
}

func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Println("sendText error:", err)
	}
}

func StartNotifications(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(time.Minute) // проверяем каждую минуту
	defer ticker.Stop()

	// Настройки для разных типов напоминаний
	reminders := []struct {
		duration   time.Duration
		statusFlag string
		messageFmt string
	}{
		{
			duration:   24 * time.Hour,
			statusFlag: "reminder24_sent",
			messageFmt: "Напоминание! Завтра в %s начнется: %s",
		},
		{
			duration:   1 * time.Hour,
			statusFlag: "reminder1_sent",
			messageFmt: "Напоминание! Через час начнется: %s",
		},
	}

	for range ticker.C {
		for _, r := range reminders {
			processReminder(bot, r.duration, r.statusFlag, r.messageFmt)
		}
		processQuorum(bot)
	}
}

func processQuorum(bot *tgbotapi.BotAPI) {
	events := db.GetUpcomingEvents(6 * time.Hour * 24)
	for _, e := range events {
		count, err := db.GetEventParticipantsCount(e.ID)
		if err != nil {
			log.Printf("Error processQuorum for event.id=%d, error: %v\n", e.ID, err)
			continue
		}
		// if registrations >= 12 send notification
		if count >= 12 {
			users, err := db.GetEventParticipants(e.ID)
			if err != nil {
				log.Printf("Error processQuorum for event.id=%d, error: %v\n", e.ID, err)
				continue
			}
			timeText := locales.FormatDateShortRU(e.StartsAt) + e.StartsAt.Format("🕐 15:04.")
			text := fmt.Sprintf(`Есть кворум!

%s
🗓 %s
📌 %s
💶 Донат на развитие клуба - 5€ с человека.

Постарайтесь не опоздать. Если что-то поменяется, обязательно напишите. Ждём! 🕵️‍♂️
`, e.Title, timeText, e.Location)

			for i, u := range users {
				text += fmt.Sprintf("%d) @%s\n", i+1, u.Nickname)
			}
			sendText(bot, int64(laVerdadChatID), text)
		}
	}
}

func processReminder(bot *tgbotapi.BotAPI, duration time.Duration, statusFlag, messageFmt string) {
	events := db.GetUpcomingEvents(duration)
	var eventIDs []string

	for _, e := range events {
		eventIDs = append(eventIDs, strconv.Itoa(e.ID))
		users := db.GetEventParticipantsWithFlag(e.ID, statusFlag)

		for _, u := range users {
			var text string
			if duration == 24*time.Hour {
				// для суток добавляем время
				text = fmt.Sprintf(messageFmt, e.StartsAt.Format("15:04 02.01.2006"), e.Title)
			} else {
				text = fmt.Sprintf(messageFmt, e.Title)
			}
			msg := tgbotapi.NewMessage(u.ChatID, text)
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения пользователю %d: %v", u.ChatID, err)
			}
		}
	}

	if len(eventIDs) > 0 {
		if err := db.UpdateRegistrationNotificationStatus(eventIDs, statusFlag); err != nil {
			log.Println(err)
		}
	}
}
