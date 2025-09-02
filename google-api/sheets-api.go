package googleapi

import (
	"context"
	"fmt"
	"laverdad-bot/db"
	"log"
	"strconv"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var service *sheets.Service
var spreadSheetID = "1fLXbsWPb7kdGGGllHLH1YI7W96fBH0Z1oZNDK0C6nvk"
var err error

func InitSheetService() {
	ctx := context.Background()

	// --- Инициализация Google Sheets API ---
	service, err = sheets.NewService(ctx, option.WithCredentialsFile("secrets/service_account.json"))
	if err != nil {
		log.Fatalf("Unable to create Sheets service: %v", err)
	}
}

func AddNewSheet(sheetName string) {
	ctx := context.Background()

	// Создаём новый лист в таблице
	addSheetReq := &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: sheetName}}
	_, err = service.Spreadsheets.BatchUpdate(spreadSheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{AddSheet: addSheetReq}},
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("Unable to add new sheet with name: %v; error:%v", sheetName, err)
	}
	log.Printf("New sheet was created: %s\n", sheetName)

	// 2. Добавляем заголовки
	rangeName := fmt.Sprintf("'%s'!A1:H1", sheetName)
	_, err = service.Spreadsheets.Values.Update(spreadSheetID, rangeName, &sheets.ValueRange{
		Values: [][]any{{"ID", "TelegramLink", "Username", "Имя", "Игровой Ник", "Статус", "CreatedAt", "UpdatedAt"}},
	}).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		log.Printf("Unable to set headers: %v", err)
	}
}

func AddRegistrationToSheet(sheetName string, line db.RegistrationLine) error {
	ctx := context.Background()

	username := fmt.Sprintf("@%s", line.UserName.String)
	rangeName := fmt.Sprintf("'%s'!A2:H2", sheetName)
	_, err = service.Spreadsheets.Values.Append(spreadSheetID, rangeName, &sheets.ValueRange{
		Values: [][]any{{line.ID, line.TelegramLink, username, line.Name, line.NickName, line.Status, line.CreatedAt.Format("02.01.2006 15:04"), line.UpdatedAt.Format("02.01.2006 15:04")}},
	}).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(ctx).Do()

	return err
}

func UpdateRegistrationStateToSheet(regID int, sheetName string, updatedAt time.Time) {
	log.Printf("Start updating google spread sheets for '%d', and event: %s\n", regID, sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadSheetID, fmt.Sprintf("'%s'!A2:A", sheetName)).Do()
	if err != nil {
		log.Printf("Unable to read data: %v", err)
	}

	rowIndex := -1

	for i, row := range resp.Values {
		log.Printf("i:%d; %v\n", i, row)

		if fmt.Sprintf("%v", row[0]) == strconv.Itoa(regID) {
			rowIndex = i + 2
			break
		}
	}

	if rowIndex == -1 {
		log.Printf("Sheets error: Registration with id=%d not found", regID)
		return
	}

	rangeName := fmt.Sprintf("'%s'!F%d:H%d", sheetName, rowIndex, rowIndex)
	vr := sheets.ValueRange{Values: [][]any{{"canceled", nil, updatedAt.Format("02.01.2006 15:04")}}}
	_, err = service.Spreadsheets.Values.Update(spreadSheetID, rangeName, &vr).ValueInputOption("RAW").Do()

	if err != nil {
		log.Printf("Unable to update status: %v", err)
	}

	log.Println("Статус обновлён на canceled")
}
