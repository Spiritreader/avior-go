package encoder

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/rs/xid"
)

type Stats struct {
	Success    bool
	Duration   int
	ExitCode   int
	OutputPath string
	Call       string
}

var state *globalstate.Data = globalstate.Instance()

func Encode(file media.File, start, duration int, overwrite bool) (Stats, error) {
	cfg := config.Instance()
	encoderConfig, ok := cfg.Local.EncoderConfig[file.Resolution.Tag]
	if !ok {
		_ = glg.Errorf("no encoder config found with tag/resolution %s/%s for %s",
			file.Resolution.Tag, file.Resolution.Value, file.Path)
		return Stats{false, -1, -1337, "", ""}, errors.New("no tag found")
	}
	params := make([]string, 0)
	if overwrite {
		params = append(params, "-y")
	} else {
		params = append(params, "-n")
	}
	for _, preArgument := range encoderConfig.PreArguments {
		split := strings.Split(preArgument, " ")
		params = append(params, split...)
	}
	if start > 0 {
		params = append(params, "-ss", strconv.Itoa(start))
	}
	params = append(params, "-i", file.Path)
	if duration > 0 {
		params = append(params, "-t", strconv.Itoa(duration))
	}
	for _, postArgument := range encoderConfig.PostArguments {
		split := strings.Split(postArgument, " ")
		params = append(params, split...)
	}
	var outPath string
	if duration > 0 && start > 0 {
		outPath = filepath.Join(filepath.Dir(file.Path), fmt.Sprintf("%s.estimate", xid.New()))
	} else {
		outPath = filepath.Join(encoderConfig.OutDirectory, file.OutName())
	}
	params = append(params, outPath)
	startTime := time.Now()
	cmd := exec.Command("ffmpeg", params...)
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	multiReader := io.MultiReader(stderr, stdout)
	if err := cmd.Start(); err != nil {
		_ = glg.Errorf("could not start ffmpeg: %s", err)
		return Stats{false, -1, -1337, "", ""}, err
	}
	scanner := bufio.NewScanner(multiReader)
	scanner.Split(ScanLinesSTDOUT)
	for scanner.Scan() {
		parseOut(scanner.Text())
	}
	if err := cmd.Wait(); err != nil {
		_ = glg.Errorf("ffmpeg error: %s", err)
	}
	exitCode := cmd.ProcessState.ExitCode()
	encTime := int(math.Round(time.Since(startTime).Minutes()))
	if exitCode != 0 {
		return Stats{false, encTime, exitCode, outPath, strings.Join(params, " ")}, nil
	}
	return Stats{true, encTime, exitCode, outPath, strings.Join(params, " ")}, nil
}

func ScanLinesSTDOUT(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0:i], nil
	} else if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a refresh that needs to be printed
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func parseOut(line string) {
	durationToken := "Duration:"
	frameToken := "frame="
	fpsToken := "fps"
	qToken := "q"
	sizeToken := "size"
	timeToken := "time"
	bitrateToken := "bitrate"
	dupToken := "dup"
	dropToken := "drop"
	speedToken := "speed"

	state.Encoder.LineOut = append(state.Encoder.LineOut, line)
	//fmt.Println(line)
	if strings.Contains(line, durationToken) {
		keyIdx := strings.Index(line, durationToken) + len(durationToken)
		timeIdx := strings.Index(line, ".")
		state.Encoder.Duration, _ = time.Parse("15:04:05", strings.Trim(line[keyIdx:timeIdx], " "))
		// safe enough for now
	} else if strings.Contains(line, frameToken) {
		// ffmpeg out looks like this:
		// frame=  272 fps=271 q=15.0 size=     512kB time=00:00:11.07 bitrate= 378.8kbits/s dup=0 drop=269 speed=  11x
		separated := make([]string, 0)
		splitEq := strings.Split(line, "=")
		for idx := range splitEq {
			splitWs := strings.Split(strings.Trim(splitEq[idx], " "), " ")
			separated = append(separated, splitWs...)
		}
		statMap := make(map[string]string)
		for idx := 0; idx < len(separated)-1; idx += 2 {
			statMap[separated[idx]] = separated[idx+1]
		}

		if val, ok := statMap[frameToken]; ok {
			frameParse, _ := strconv.ParseInt(val, 10, 32)
			state.Encoder.Frame = int(frameParse)
		}
		if val, ok := statMap[fpsToken]; ok {
			state.Encoder.Fps, _ = strconv.ParseFloat(val, 64)
		}
		if val, ok := statMap[qToken]; ok {
			state.Encoder.Q, _ = strconv.ParseFloat(val, 64)
		}
		if val, ok := statMap[sizeToken]; ok {
			state.Encoder.Size = val
		}
		if val, ok := statMap[timeToken]; ok {
			cutIdx := strings.Index(val, ".")
			if cutIdx != -1 {
				state.Encoder.Position, _ = time.Parse("15:04:05", val)
			}
		}
		if val, ok := statMap[bitrateToken]; ok {
			state.Encoder.Bitrate = val
		}
		if val, ok := statMap[dupToken]; ok {
			dupParse, _ := strconv.ParseInt(val, 10, 32)
			state.Encoder.Dup = int(dupParse)
		}
		if val, ok := statMap[dropToken]; ok {
			dropParse, _ := strconv.ParseInt(val, 10, 32)
			state.Encoder.Drop = int(dropParse)
		}
		if val, ok := statMap[speedToken]; ok {
			cutIdx := strings.Index(val, "x")
			if cutIdx != -1 {
				state.Encoder.Speed, _ = strconv.ParseFloat(strings.Trim(val[:cutIdx], " "), 64)
			}
		}

		if state.Encoder.Speed > 0 {
			// calculate ETA
			diff := state.Encoder.Duration.Sub(state.Encoder.Position)
			diff /= time.Duration(state.Encoder.Speed)
			state.Encoder.Remaining = diff
		}
	}
	fmt.Printf("Duration: %s ", state.Encoder.Duration.Format("15:04:05"))
	fmt.Printf("Frame: %d ", state.Encoder.Frame)
	fmt.Printf("Fps: %.2f ", state.Encoder.Fps)
	fmt.Printf("Q: %.0f ", state.Encoder.Q)
	fmt.Printf("Size: %s ", state.Encoder.Size)
	fmt.Printf("Position: %s ", state.Encoder.Duration.Format("15:04:05"))
	fmt.Printf("Bitrate: %s ", state.Encoder.Bitrate)
	fmt.Printf("Dup: %d ", state.Encoder.Dup)
	fmt.Printf("Drop: %d ", state.Encoder.Drop)
	fmt.Printf("Speed: %.1f ", state.Encoder.Speed)
	fmt.Printf("Remaining: %s\n", state.Encoder.Remaining)
}
