package joblog

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Data struct {
	messages []string
}

func (j *Data) Add(line string) {
	if j.messages == nil {
		j.messages = make([]string, 0)
	}
	j.messages = append(j.messages, line)
}

func (j *Data) AddFileProperties(file media.File) {
	if j.messages == nil {
		j.messages = make([]string, 0)
	}
	j.messages = append(j.messages, fmt.Sprintf("OriginalPath: %s", file.Path))
	j.messages = append(j.messages, fmt.Sprintf("Recorded/Length: %dm/%dm", file.RecordedLength, file.Length))
	j.messages = append(j.messages, fmt.Sprintf("Audio: %s", file.AudioFormat.String()))
	j.messages = append(j.messages, fmt.Sprintf("EncodeParams: %s", file.CustomParams))
}

func (j *Data) AppendTo(path string, newline bool, separators bool) error {
	// lumberjack opens log files with 0600. Create the path first so a new
	// file gets the required mode, and fix existing files before rotation.
	if err := ensureLogFilePermissions(path); err != nil {
		_ = glg.Warnf("could not prepare log file %s: %s", path, err)
	}
	file := &lumberjack.Logger{
		Filename: path,
		MaxSize:  10, // megabytes
		//MaxBackups: 3,
		//MaxAge:     28,   //days
		//Compress:   false, // disabled by default
	}
	defer func() {
		err := file.Close()
		if err != nil {
			_ = glg.Errorf("error closing file: %s", err)
		}
	}()
	hostname, _ := os.Hostname()
	writer := bufio.NewWriter(file)
	if newline {
		_, _ = writer.WriteString("\n")
	}
	if separators {
		_, _ = writer.WriteString("----------------\n")
	} else {
		_, _ = writer.WriteString(consts.LOG_DELIM + "\n")
	}
	_, _ = writer.WriteString(fmt.Sprintf("%s - %s \n", hostname, time.Now().Format("Monday 2006-01-02 15:04:05 -0700 MST")))
	_, _ = writer.WriteString("\n")
	for _, message := range j.messages {
		_, _ = writer.WriteString(message + "\n")
	}
	if separators {
		_, _ = writer.WriteString("----------------\n\n")
	}
	if err := writer.Flush(); err != nil {
		_ = glg.Errorf("could not write media info to %s, err: %s", path, err)
		return err
	}
	// A new file may have been opened by lumberjack during this write.
	if err := ensureLogFilePermissions(path); err != nil {
		_ = glg.Errorf("could not set permissions on media info log %s, err: %s", path, err)
		return err
	}
	return nil
}

func ensureLogFilePermissions(path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0666)
}
