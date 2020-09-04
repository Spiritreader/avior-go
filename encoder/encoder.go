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
}

var state *globalstate.Data = globalstate.Instance()

func Encode(file media.File, start, duration int, overwrite bool) (Stats, error) {
	cfg := config.Instance()
	encoderConfig, ok := cfg.Local.EncoderConfig[file.Resolution.Tag]
	if !ok {
		_ = glg.Errorf("no encoder config found with tag/resolution %s/%s for %s",
			file.Resolution.Tag, file.Resolution.Value, file.Path)
		return Stats{false, -1, -1337, ""}, errors.New("no tag found")
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
		outPath = filepath.Join(filepath.Dir(file.Path), fmt.Sprintf("%s.mkv.estimate", xid.New()))
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
		return Stats{false, -1, -1337, ""}, err
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
		return Stats{false, encTime, exitCode, outPath}, nil
	}
	return Stats{true, encTime, exitCode, outPath}, nil
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
	fpsToken := "fps="
	qToken := "q="
	sizeToken := "size="
	timeToken := "time="
	bitrateToken := "bitrate="
	dupToken := "dup="
	dropToken := "drop="
	speedToken := "speed="

	state.Encoder.LineOut = append(state.Encoder.LineOut, line)
	if strings.Contains(line, durationToken) {
		keyIdx := strings.Index(line, durationToken) + len(durationToken)
		timeIdx := strings.Index(line, ".")
		state.Encoder.Duration, _ = time.Parse("15:04:05", strings.Trim(line[keyIdx:timeIdx], " "))
	// safe enough for now
	} else if strings.Contains(line, frameToken) {
		// ffmpeg out looks like this: 
		// frame=  272 fps=271 q=15.0 size=     512kB time=00:00:11.07 bitrate= 378.8kbits/s dup=0 drop=269 speed=  11x
		
		next := strings.Split(line, frameToken)
		// next[0] empty 


		next = strings.Split(next[1], fpsToken)
		// next[0] now holds frame value
		frameParse, _ := strconv.ParseInt(strings.Trim(next[0], " "), 10, 32)
		state.Encoder.Frame = int(frameParse)


		next = strings.Split(next[1], qToken)
		// next[0] now holds fps value
		state.Encoder.Fps, _ = strconv.ParseFloat(strings.Trim(next[0], " "), 64)


		next = strings.Split(next[1], sizeToken)
		// next[0] now holds q value
		state.Encoder.Q, _ = strconv.ParseFloat(strings.Trim(next[0], " "), 64)


		next = strings.Split(next[1], timeToken)
		// next[0] now holds size value
		state.Encoder.Size = strings.Trim(next[0], " ")


		next = strings.Split(next[1], bitrateToken)
		// next[0] now time value
		state.Encoder.Position, _ = time.Parse("15:04:05", strings.Trim(next[0][:strings.Index(next[0], ".")], " "))


		next = strings.Split(next[1], dupToken)
		// next[0] now holds bitrate value
		state.Encoder.Bitrate = strings.Trim(next[0], " ")


		next = strings.Split(next[1], dropToken)
		// next[0] now holds dup value
		dupParse, _ := strconv.ParseInt(strings.Trim(next[0], " "), 10, 32)
		state.Encoder.Dup = int(dupParse)


		next = strings.Split(next[1], speedToken)
		// next[1] now holds speed value
		state.Encoder.Speed, _ = strconv.ParseFloat(strings.Trim(next[1][:strings.Index(next[1], "x")], " "), 64)
	}

	fmt.Printf("Duration: %s\n", state.Encoder.Duration)
	fmt.Printf("Frame: %d\n", state.Encoder.Frame)
	fmt.Printf("Fps: %f\n", state.Encoder.Fps)
	fmt.Printf("Q: %f\n", state.Encoder.Q)
	fmt.Printf("Size: %s\n", state.Encoder.Size)
	fmt.Printf("Position: %s\n", state.Encoder.Position)
	fmt.Printf("Bitrate: %s\n", state.Encoder.Bitrate)
	fmt.Printf("Dup: %d\n", state.Encoder.Dup)
	fmt.Printf("Drop: %d\n", state.Encoder.Drop)
	fmt.Printf("Speed: %f\n", state.Encoder.Speed)
}