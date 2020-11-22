package encoder

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/rs/xid"
	"golang.org/x/sys/windows"
)

type Stats struct {
	Success  bool
	Duration time.Duration
	ExitCode int
	// Encoded output path including file name
	OutputPath string
	Call       string
}

var state *globalstate.Data = globalstate.Instance()

func Encode(file media.File, start, duration int, overwrite bool, dstDir *string) (Stats, error) {
	state.Encoder.Active = true
	state.Encoder.LineOut = make([]string, 0)
	defer func() {
		state.Encoder.Active = false
	}()
	cfg := config.Instance()
	encoderConfig, ok := cfg.Local.EncoderConfig[file.Resolution.Tag]
	if !ok {
		_ = glg.Errorf("no encoder config found for %s", file.Path)
		return Stats{false, -1, -1337, "", ""}, errors.New("no tag found")
	}
	_ = glg.Infof("tag/resolution %s:%s", file.Resolution.Tag, file.Resolution.Value)

	// use custom parameters instead of encoder config if provided
	if len(file.CustomParams) > 0 {
		customPreArgs := make([]string, 0)
		customPostArgs := make([]string, 0)
		for _, cParam := range file.CustomParams {
			if strings.HasPrefix(cParam, consts.COMPAT_CPARAM_PREFIX) {
				split := strings.Split(cParam, consts.COMPAT_CPARAM_PREFIX)
				customPreArgs = append(customPreArgs, split[1])
			} else {
				customPostArgs = append(customPostArgs, cParam)
			}
		}
		encoderConfig.PreArguments = customPreArgs
		encoderConfig.PostArguments = customPostArgs
	}

	// allow overwrite setting
	params := make([]string, 0)
	if overwrite {
		params = append(params, "-y")
	} else {
		params = append(params, "-n")
	}

	// pre arguments for ffmpeg
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

	// post arguments for ffmpeg
	for _, postArgument := range encoderConfig.PostArguments {
		split := strings.Split(postArgument, " ")
		params = append(params, split...)
	}

	// determine which output path to use
	customDuration := false
	var outPath string
	if duration > 0 && start > 0 {
		outPath = filepath.Join(filepath.Dir(file.Path), fmt.Sprintf("%s.estimate.mkv", xid.New()))
		durationTime := new(time.Time).Add(time.Duration(duration)*time.Second).AddDate(-1, 0, 0)
		state.Encoder.Duration = durationTime
		customDuration = true
		_ = glg.Infof("output file path: %s", outPath)
	} else if dstDir != nil {
		outPath = filepath.Join(*dstDir, file.OutName()+cfg.Local.Ext)
		_ = glg.Infof("output file path: %s", outPath)
	} else {
		outPath = filepath.Join(encoderConfig.OutDirectory, file.OutName()+cfg.Local.Ext)
		_ = glg.Infof("output file path: %s", outPath)
	}
	state.Encoder.OutPath = outPath

	// call ffmpeg
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

	hProcess, err := windows.OpenProcess(0x0400|0x0200, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = glg.Warnf("could not get ffmpeg handle using pid %d, err: %s", cmd.Process.Pid, err)
	}
	err = windows.SetPriorityClass(hProcess, config.PriorityUint32(cfg.Local.EncoderPriority))
	if err != nil {
		_ = glg.Warnf("could not set priority %s for ffmpeg handle using pid %d, err: %s",
			cfg.Local.EncoderConfig, cmd.Process.Pid, err)
	}
	err = windows.CloseHandle(hProcess)
	if err != nil {
		_ = glg.Errorf("could not close handle for pid %d, err: %s", cmd.Process.Pid, err)
	}

	// scan stdout
	scanner := bufio.NewScanner(multiReader)
	scanner.Split(ScanLinesSTDOUT)
	for scanner.Scan() {
		parseOut(scanner.Text(), customDuration)
	}
	if err := cmd.Wait(); err != nil {
		_ = glg.Errorf("ffmpeg error: %s", err)
	}
	exitCode := cmd.ProcessState.ExitCode()
	encTime := time.Since(startTime)
	if exitCode != 0 {
		return Stats{false, encTime, exitCode, outPath, strings.Join(params, " ")}, errors.New("exit code not ok")
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

func parseOut(line string, customDuration bool) {
	durationToken := "Duration:"
	frameToken := "frame"
	fpsToken := "fps"
	qToken := "q"
	sizeToken := "size"
	timeToken := "time"
	bitrateToken := "bitrate"
	dupToken := "dup"
	dropToken := "drop"
	speedToken := "speed"

	state.Encoder.LineOut = append(state.Encoder.LineOut, line)
	if !customDuration && strings.Contains(line, durationToken) {
		//fmt.Println(line)
		_ = glg.Log(line)
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
			//fmt.Printf("Duration: %s\n", state.Encoder.Duration)
			//fmt.Printf("Position: %s\n", state.Encoder.Position)
			diff := state.Encoder.Duration.Sub(state.Encoder.Position)
			//fmt.Printf("Difference: %s\n", diff)
			if speed := time.Duration(state.Encoder.Speed); speed > 0 {
				diff /= speed
			}
			state.Encoder.Remaining = diff
		}
		durationDuration := state.Encoder.Duration.Sub(new(time.Time).AddDate(-1, 0, 0)).Seconds()
		positionDuration := state.Encoder.Position.Sub(new(time.Time).AddDate(-1, 0, 0)).Seconds()
		state.Encoder.Progress = (positionDuration / durationDuration) * 100
		termOut := ""
		termOut += fmt.Sprintf("Duration: %s ", state.Encoder.Duration.Format("15:04:05"))
		termOut += fmt.Sprintf("Frame: %d ", state.Encoder.Frame)
		termOut += fmt.Sprintf("Fps: %.2f ", state.Encoder.Fps)
		termOut += fmt.Sprintf("Q: %.0f ", state.Encoder.Q)
		termOut += fmt.Sprintf("Size: %s ", state.Encoder.Size)
		termOut += fmt.Sprintf("Position: %s ", state.Encoder.Position.Format("15:04:05"))
		termOut += fmt.Sprintf("Bitrate: %s ", state.Encoder.Bitrate)
		termOut += fmt.Sprintf("Dup: %d ", state.Encoder.Dup)
		termOut += fmt.Sprintf("Drop: %d ", state.Encoder.Drop)
		termOut += fmt.Sprintf("Speed: %.1f ", state.Encoder.Speed)
		termOut += fmt.Sprintf("Remaining: %s", state.Encoder.Remaining)
		_ = glg.Log(termOut)
		//fmt.Printf("\r" + termOut)
	}
}
