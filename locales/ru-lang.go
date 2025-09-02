package locales

import (
	"fmt"
	"time"
)

var ruMonths = []string{
	"января", "февраля", "марта", "апреля", "майя", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}
var ruWeekdays = []string{
	"воскресенье", "понедельник", "вторник", "среда",
	"четверг", "пятница", "суббота",
}

var ruWeekdaysShort = []string{
	"вс", "пн", "вт", "ср", "чт", "пт", "сб",
}

func FormatDateRU(t time.Time) string {
	day := t.Day()
	month := ruMonths[int(t.Month())-1]
	weekday := ruWeekdays[int(t.Weekday())]
	return fmt.Sprintf("%s, %d %s", weekday, day, month)
}

func FormatDateShortRU(t time.Time) string {
	day := t.Day()
	month := ruMonths[int(t.Month())-1]
	weekday := ruWeekdaysShort[int(t.Weekday())]
	return fmt.Sprintf("%d %s (%s)", day, month, weekday)
}
