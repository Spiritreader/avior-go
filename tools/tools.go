package tools

import (
	"fmt"
	"strings"
	"time"
)

//InTimeSpan(start, end, check) determines if check lies between start and end
func InTimeSpan(startString string, endString string, checkTime time.Time) bool {
	if startString == endString {
		return true
	}
	layout := "15:04"
	start, _ := time.Parse(layout, startString)
	end, _ := time.Parse(layout, endString)
	check, _ := time.Parse(layout, fmt.Sprintf("%d:%d", checkTime.Hour(), checkTime.Minute()))
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func RemoveIllegalChars(str string) string {
	toWhitespace := "\\<>|"
	toNone := ":?*\""
	toUnderscore := "/"
	for _, rune := range toWhitespace {
		str = strings.ReplaceAll(str, string(rune), " ")
	}
	for _, rune := range toNone {
		str = strings.ReplaceAll(str, string(rune), "")
	}
	for _, rune := range toUnderscore {
		str = strings.ReplaceAll(str, string(rune), "_")
	}
	return str
}
