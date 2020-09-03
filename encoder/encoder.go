package encoder

import (
	"bufio"
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
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := cmd.Wait(); err != nil {
		_ = glg.Errorf("waiting for ffmpeg failed: %s", err)
		return Stats{false, -1, -1337, outPath}, err
	}
	exitCode := cmd.ProcessState.ExitCode()
	encTime := int(math.Round(time.Since(startTime).Minutes()))
	if exitCode != 0 {
		return Stats{false, encTime, exitCode, outPath}, nil
	}
	return Stats{true, encTime, exitCode, outPath}, nil
}
