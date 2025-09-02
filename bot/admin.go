package bot

import (
	"fmt"
	"time"

	"laverdad-bot/db"
	googleapi "laverdad-bot/google-api"
	"laverdad-bot/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var adminIDs = map[int64]bool{
	115775166: true,
	107463316: true,
	102833932: true,
}

type AdminState struct {
	Step      string
	TempEvent db.Event
}

var adminStates = map[int64]*AdminState{}

func IsAdmin(userID int64) bool {
	return adminIDs[userID]
}

func HandleAdmin(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if !IsAdmin(msg.From.ID) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Нет доступа"))
		return
	}

	state, ok := adminStates[msg.From.ID]
	if !ok {
		state = &AdminState{}
		adminStates[msg.From.ID] = state
	}

	switch state.Step {
	case "":
		switch msg.Text {
		case "/notify_registration":
			services.NotifyRegistrationStarted(bot)
		case "/generate":
			services.CreateFridayEvent()
			services.CreateSaturdayEvent()
			services.CreateSundayEvent()
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ События успешно созданы!"))
		case "/addevent":
			state.Step = "title"
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Введите заголовок события:"))
		case "/registrations":
			events := db.GetEvents()
			if len(events) == 0 {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Пока нет доступных событий."))

				return
			}

			text := "Список мероприятий:\n"
			markup := tgbotapi.NewInlineKeyboardMarkup()
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, e := range events {
				row := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("%s — %s", e.Title, e.StartsAt.Format("02.01 15:04")),
						fmt.Sprintf("admin_ev_%d", e.ID),
					),
				)
				rows = append(rows, row)
			}
			markup.InlineKeyboard = rows
			msg := tgbotapi.NewMessage(msg.Chat.ID, text)
			msg.ReplyMarkup = markup
			bot.Send(msg)
		default:
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Доступные команды:\n/addevent\n/registrations\n/generate\n/notify_registration"))
		}
	case "title":
		state.TempEvent.Title = msg.Text
		state.Step = "description"
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Введите описание события:"))
	case "description":
		state.TempEvent.Description = msg.Text
		state.Step = "location"
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Введите место проведения события:"))
	case "location":
		state.TempEvent.Location = msg.Text
		state.Step = "datetime"
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Введите дату и время события в формате 2006-01-02 15:04:"))
	case "datetime":
		dt, err := time.Parse("2006-01-02 15:04", msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неверный формат, попробуйте ещё раз:"))
			return
		}
		state.TempEvent.StartsAt = dt
		db.CreateEvent(state.TempEvent)
		go googleapi.AddNewSheet(fmt.Sprintf("%s - %s", state.TempEvent.Title, state.TempEvent.StartsAt.Format("02.01")))
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Событие добавлено!"))
		state.Step = ""
	}
}
