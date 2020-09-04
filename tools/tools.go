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

func ByteCountUpSI(b int64, upBy int) (float64, string) {
	const unit = 1000
	upBy--
	if b < unit || upBy < 1 {
		return float64(b), fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for exp < upBy {
		div *= unit
		exp++
	}
	outVal := float64(b) / float64(div)
	return outVal, fmt.Sprintf("%.1f %ciB",
	float64(b) / float64(div), "KMGTPE"[exp])
}

func ByteCountDownSI(b float64, exp int, downBy int) (float64, string) {
	const unit = 1000
	prefixes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

	if downBy >= exp {
		downBy = exp-1
	}
	if downBy == 0 {
		return b, fmt.Sprintf("%.1f %s", b, prefixes[exp-1])
	}
	mul := int64(unit)
	for i := 0; i < downBy - 1; i++ {
		mul *= unit
	}
	outVal := float64(b) * float64(mul)
	return outVal, fmt.Sprintf("%.1f %s",
	float64(b) * float64(mul), prefixes[exp-downBy-1])
}

func ByteCountSI(b int64) string {
    const unit = 1000
    if b < unit {
        return fmt.Sprintf("%d B", b)
    }
    div, exp := int64(unit), 0
    for n := b / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB",
        float64(b)/float64(div), "kMGTPE"[exp])
}