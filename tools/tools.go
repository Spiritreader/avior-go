package tools

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/kpango/glg"
)

var ZeroDurationError error = fmt.Errorf("duration is 0")
var NoStreamsError error = fmt.Errorf("no streams found")

//InTimeSpan(start, end, check) determines if check lies between start and end
func InTimeSpan(startString string, endString string, checkTime time.Time) bool {
	if startString == endString {
		return true
	}
	layout := "15:04"
	start, _ := time.Parse(layout, startString)
	end, _ := time.Parse(layout, endString)
	check, _ := time.Parse(layout, fmt.Sprintf("%02d:%02d", checkTime.Hour(), checkTime.Minute()))
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
		return float64(b), fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for exp < upBy {
		div *= unit
		exp++
	}
	outVal := float64(b) / float64(div)
	return outVal, fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func ByteCountDownSI(b float64, exp int, downBy int) (float64, string) {
	const unit = 1000
	prefixes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

	if downBy >= exp {
		downBy = exp - 1
	}
	if downBy == 0 {
		return b, fmt.Sprintf("%.1f %s", b, prefixes[exp-1])
	}
	mul := int64(unit)
	for i := 0; i < downBy-1; i++ {
		mul *= unit
	}
	outVal := float64(b) * float64(mul)
	return outVal, fmt.Sprintf("%.1f %s",
		float64(b)*float64(mul), prefixes[exp-downBy-1])
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// interface wrapper for io.Reader
type PassThru struct {
	io.Reader
	transferred int64
	totalBytes  int64
	data        *globalstate.Data
	name        string
}

// Gets the channel layout and number of channels from a file using ffprobe
// Returns the number of channels, the channel layout and an error if one occurs
func FfProbeChannels(path string) (int, string, error) {
	// open ffprobe at the specified path via cmd.exec, get the number of audio channels from the output and save into an integer
	channels := -1
	channel_layout := "unknown"
	ffprobe := exec.Command("ffprobe", "-v", "quiet", "-select_streams", "a:0", "-show_entries", "stream=channels, channel_layout", "-of", "default=noprint_wrappers=1", path)
	output, err := ffprobe.Output()
	if err != nil {
		return channels, channel_layout, err
	}
	data := strings.Split(string(output), "\n")

	if len(data) > 1 {
		channels_string := strings.Trim(strings.Split(data[0], "=")[1], "\r ")
		channel_layout = strings.Trim(strings.Split(data[1], "=")[1], "\r ")
		channels, err = strconv.Atoi(channels_string)
		if err != nil {
			return channels, channel_layout, err
		}
	}
	return channels, channel_layout, nil
}

func FfProbeVerfiy(path string) (bool, error) {
	ffprobe := exec.Command("ffprobe", "-v", "quiet", "-select_streams", "v", "-show_entries", "stream_tags=duration", "-of", "default=noprint_wrappers=1", path)
	output, err := ffprobe.Output()
	if err != nil {
		return true, err
	}
	if (len(output) == 0) {
		return false, NoStreamsError
	}
	data := strings.Split(string(output), "\n")
	if len(data) > 1 {
		durationString := strings.Trim(strings.Split(data[0], "=")[1], "\r ")
		duration, err := ParseDuration(durationString)
		if err != nil {
			return true, err
		}

		if duration == 0 {
			return false, ZeroDurationError
		}
	}
	return true, nil
}

func ParseDuration(durationStr string) (time.Duration, error) {
	parts := strings.SplitN(durationStr, ".", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid duration format, no fraction")
	}

	hms := strings.Split(parts[0], ":")
	if len(hms) != 3 {
		return 0, fmt.Errorf("invalid duration format, no hours, minutes or seconds")
	}

	hours, err := strconv.Atoi(hms[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(hms[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(hms[2])
	if err != nil {
		return 0, err
	}

	fractionStr := parts[1]
	fraction, err := strconv.ParseInt(fractionStr, 10, 64)
	if err != nil {
		return 0, err
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(fraction)

	return duration, nil
}

func MoppyFile(src string, dst string, move bool) error {

	if move {
		_ = glg.Infof("moving %s to %s", src, dst)
		err := os.Rename(src, dst)
		if err != nil {
			_ = glg.Warnf("could not move file directly: %s", err)
		} else {
			return nil
		}
	} else {
		_ = glg.Infof("copy-ying %s to %s", src, dst)
	}

	state := globalstate.Instance()
	state.Mover.Active = true
	defer func() {
		state.Mover.Active = false
	}()
	source, err := os.Open(src)
	if err != nil {
		_ = glg.Errorf("could not open file: %s", err)
		return err
	}
	sourceInfo, err := os.Stat(src)
	if err != nil {
		_ = glg.Errorf("could not get metadata from file: %s", err)
		return err
	}
	destination, err := os.Create(dst)
	if err != nil {
		_ = glg.Errorf("could not create destination file: %s", err)
		return err
	}
	var reader io.Reader
	reader = source
	reader = &PassThru{
		Reader:     reader,
		data:       state,
		totalBytes: sourceInfo.Size(),
		name:       filepath.Base(dst),
	}
	_, err = io.Copy(destination, reader)
	_ = source.Close()
	_ = destination.Close()
	if err != nil {
		err = os.Remove(dst)
		if err != nil {
			_ = glg.Errorf("failing to remove destination file while failing to copy \"%s\": %s", src, err)
			return err
		}
		_ = glg.Errorf("could not copy file \"%s\" to \"%s\": %s", src, dst, err)
		return err
	}
	if move {
		err = os.Remove(src)
		if err != nil {
			_ = glg.Errorf("could not remove source file: %s", err)
			return err
		}
	}
	return nil
}

func (pt *PassThru) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)
	if err == nil {
		pt.transferred += int64(n)
		pt.data.Mover.Progress = int((float64(pt.transferred) / float64(pt.totalBytes)) * 100)
		pt.data.Mover.Position = ByteCountSI(pt.transferred)
		pt.data.Mover.FileSize = ByteCountSI(pt.totalBytes)
		if pt.transferred%100000 == 0 {
			//fmt.Printf("\rFile: %s Bytes: %s Total: %s", pt.name, ByteCountSI(pt.transferred), ByteCountSI(pt.totalBytes))
			fmt.Printf("File: %s Bytes: %s Total: %s\n", pt.name, ByteCountSI(pt.transferred), ByteCountSI(pt.totalBytes))
		}
	}
	return n, err
}

const TH32CS_SNAPPROCESS = 0x00000002

type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}
