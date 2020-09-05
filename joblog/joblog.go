package joblog

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
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

func (j* Data) AddFileProperties(file media.File) {
	if j.messages == nil {
		j.messages = make([]string, 0)
	}
	j.messages = append(j.messages, fmt.Sprintf("Original Path: %s", file.Path))
	j.messages = append(j.messages, fmt.Sprintf("Recorded/Length: %dm/%dm", file.RecordedLength, file.Length))
	j.messages = append(j.messages, fmt.Sprintf("Audio: %s", file.AudioFormat.String()))
	j.messages = append(j.messages, fmt.Sprintf("EncodeParams: %s", file.EncodeParams))
}

func (j *Data) AppendTo(path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		err := file.Close()
		if err != nil {
			_ = glg.Errorf("error closing file: %s", err)
		}
	}()
	if err != nil {
		_ = glg.Errorf("could not write media info to %s, err: %s", path, err)
		return err
	}
	hostname, _ := os.Hostname()
	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString("----------------\n")
	_, _ = writer.WriteString(fmt.Sprintf("%s - %s \n", hostname, time.Now()))
	_, _ = writer.WriteString("\n")
	for _, message := range j.messages {
		_, _ = writer.WriteString(message + "\n")
	}
	_, _ = writer.WriteString("\n")
	_, _ = writer.WriteString("----------------\n")
	err = writer.Flush()
	if err != nil {
		_ = glg.Errorf("could not write media info to %s, err: %s", path, err)
		return err
	}
	return nil
}