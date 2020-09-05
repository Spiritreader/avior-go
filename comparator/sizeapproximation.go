package comparator

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/encoder"
	"github.com/Spiritreader/avior-go/globalstate"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

var state *globalstate.Data = globalstate.Instance()

type SizeApproxModule struct {
	moduleConfig *structs.ModuleConfig
	settings     structs.SizeApproxModuleSettings
	new          media.File
	duplicate    media.File
}

func (s *SizeApproxModule) Init(mcfg structs.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *SizeApproxModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &structs.SizeApproxModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	s.new = files[0]
	s.duplicate = files[0]
	startTime := time.Now()
	estimatedSize, duplicateSize, difference, err := s.estimate()
	state.Encoder.Slice = 0
	state.Encoder.OfSlices = 0
	if err != nil {
		return s.Name(), NOCH, fmt.Sprintf("module error: %s", err)
	}
	since := time.Since(startTime)
	_ = glg.Infof("approx module: estimation took %s", since)
	if difference > s.settings.Difference {
		return s.Name(), REPL, fmt.Sprintf("new/old: ratio: %s/%s: %d",
			tools.ByteCountSI(estimatedSize), tools.ByteCountSI(duplicateSize), difference)
	} else if difference >= 0 {
		return s.Name(), NOCH, fmt.Sprintf("new/old: ratio: %s/%s: %d",
			tools.ByteCountSI(estimatedSize), tools.ByteCountSI(duplicateSize), difference)
	} else if difference < 0 {
		return s.Name(), KEEP, fmt.Sprintf("new/old: ratio: %s/%s: %d",
			tools.ByteCountSI(estimatedSize), tools.ByteCountSI(duplicateSize), difference)
	}
	return s.Name(), KEEP, "no criteria matched for replacement"
}

func (s *SizeApproxModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *SizeApproxModule) Name() string {
	return consts.MODULE_NAME_SIZEAPPROX
}

// esimate returns (estimatedSize, duplicateSize, differenceFraction, err)
func (s *SizeApproxModule) estimate() (int64, int64, int, error) {
	if s.settings.Difference <= 0 || s.settings.SampleCount <= 0 && s.settings.Fraction <= 0 {
		_ = glg.Error("invalid module settings, must be greater than 0")
		return -1, -1, -1, errors.New("invalid settings")
	}
	encSlices := s.settings.SampleCount
	state.Encoder.OfSlices = encSlices
	// Time units is how many seconds slices are apart from each other
	// encSlices + 1 because there needs to be room at the end of the file and the entry point mustn't be EOF for 1 slice.
	timeUnits := s.duplicate.RecordedLength * 60 / (encSlices + 1)
	samples := make([]int64, s.settings.SampleCount)

	// Start at half the time unit to avoid hitting opening sequences / black screens in movies
	position := timeUnits / 2
	secondsPerEncSlice := int(math.Ceil(float64(s.duplicate.RecordedLength) * float64(s.settings.Fraction) * 0.01 * 60))
	if secondsPerEncSlice < 60 {
		secondsPerEncSlice = 60
	}
	// If the total encoding time in minutes is greater or equal to size, encode the whole thing
	encodingDuration := time.Second * time.Duration(secondsPerEncSlice) * time.Duration(encSlices)
	if encodingDuration >= time.Minute*time.Duration(s.duplicate.RecordedLength) {
		encSlices = 1
		position = 0
		timeUnits = 0
		secondsPerEncSlice = s.duplicate.RecordedLength * 60
	} else if secondsPerEncSlice > timeUnits {
		_ = glg.Warnf("overlap detected with slice length %d, adjusted seconds per slice to %d", secondsPerEncSlice, timeUnits)
		secondsPerEncSlice = timeUnits
	}

	// encode all slices and get sample file sizes
	for idx := range samples {
		state.Encoder.Slice = idx
		stats, err := encoder.Encode(s.duplicate, position, secondsPerEncSlice, true)
		if err != nil {
			_ = glg.Errorf("error encoding %s for estimation, output path %s, err: %s",
				s.duplicate.Path, stats.OutputPath, err)
			return -1, -1, -1, err
		}
		outFile, err := os.Stat(stats.OutputPath)
		if err != nil {
			_ = glg.Errorf("could not open estimation file %s, err: %s", stats.OutputPath, err)
		}
		samples[idx] = outFile.Size()
		err = os.Remove(stats.OutputPath)
		if err != nil {
			_ = glg.Warnf("could not delete estimation file %s, err: %s", stats.OutputPath, err)
		}
		// advance position for each slice
		position += timeUnits
	}

	var avg float64
	for _, sample := range samples {
		avg += float64(sample)
	}
	// avg slice size
	avg /= float64(encSlices)
	// avg slice size per minute
	avg /= float64(secondsPerEncSlice)
	avg *= 60
	estimatedFileSize := int64(math.Ceil(avg * float64(s.duplicate.RecordedLength)))
	duplicateFileSize, err := os.Stat(s.duplicate.Path)
	if err != nil {
		_ = glg.Errorf("could not read original file %s, err: %s", s.duplicate.Path, err)
		return -1, -1, -1, err
	}
	difference := 100 - ((estimatedFileSize / duplicateFileSize.Size()) * 100)
	return estimatedFileSize, duplicateFileSize.Size(), int(difference), nil
}
