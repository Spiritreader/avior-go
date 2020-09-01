package media

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
)

/*
Probability values:
> 0 MULTI, < 0 STEREO
None in log: 0
STEREO only in tuner: -3
MULTI only in tuner: +3
MULTI + STEREO in log, STEREO tag in meta: -2
MULTI + STEREO in log, No Tag: +1
MULTI + STEREO in log, MULTI tag in meta: +2
Probability High: Only tuner
Probability Med: tuner + tags
Probability Low: tuner + meta without tag
*/ //
const (
	STEREO          = -3
	STEREO_PROBABLY = -2
	STEREO_MAYBE    = -1
	AUDIO_UNKNOWN   = 0
	MULTI_MAYBE     = 1
	MULTI_PROBABLY  = 2
	MULTI           = 3
)

type Resolution struct {
	Tag   string
	Value string
}

func (r *Resolution) GetPixels() (int64, error) {
	strSlice := strings.Split(r.Value, "x")
	intSlice := make([]int64, len(strSlice))
	for idx, str := range strSlice {
		i, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			_ = glg.Errorf("could not convert resolution %s to pixel value: %s", r.Value, err)
			return 0, err
		}
		intSlice[idx] = i
	}
	var out int64 = 1
	for _, dim := range intSlice {
		out *= dim
	}
	return out, nil
}

type FileInfo struct {
	Path       string
	Name       string
	Subtitle   string
	Resolution Resolution
	// duration the tuner spent recording this file
	RecordedLength int
	// duration provided by epg
	Length int
	//
	// Probability values:
	//
	// > 0 MULTI, < 0 STEREO
	//
	// None in log: 0
	//
	// STEREO only in tuner: -3
	//
	// MULTI only in tuner: +3
	//
	// MULTI + STEREO in log, STEREO tag in meta: -2
	//
	// MULTI + STEREO in log, No Tag: +1
	//
	// MULTI + STEREO in log, MULTI tag in meta: +2
	//
	// Probability High: Only tuner
	//
	// Probability Med: tuner + tags
	//
	// Probability Low: tuner + meta without tag
	AudioFormat int
	MetadataLog []string
	TunerLog    []string
	LogPaths    []string
	legacy      bool
}

// Updates the struct to fill out all remaining fields
func (f *FileInfo) Update() error {
	if err := f.readLogs(); err != nil {
		return err
	}
	f.getAudio()
	f.getResolution()
	f.getLength()
	return nil
}

// getAudio retrieves the audio file from the log files and updates the struct
func (f *FileInfo) getAudio() {
	cfg := config.Instance()

	tunerStereo := Find(f.TunerLog, cfg.Local.AudioFormats.StereoTags)
	tunerMulti := Find(f.TunerLog, cfg.Local.AudioFormats.MultiTags)

	if tunerStereo && !tunerMulti {
		// guaranteed to be stereo if tuner only picks up one audio codec
		f.AudioFormat = STEREO
	} else if !tunerStereo && tunerMulti {
		// guaranteed to be multichannel if tuner only picks up one audio codec
		f.AudioFormat = MULTI
	} else if tunerStereo && tunerMulti {
		// complement info with tags if available
		metaStereo := Find(f.MetadataLog, cfg.Local.AudioFormats.StereoTags)
		metaMulti := Find(f.MetadataLog, cfg.Local.AudioFormats.MultiTags)

		if metaMulti {
			// if tags include multichannel audio, it's still likely to be multichannel
			f.AudioFormat = MULTI_PROBABLY
		} else if metaStereo {
			f.AudioFormat = STEREO_PROBABLY
		} else if !metaMulti && !metaStereo {
			f.AudioFormat = MULTI_MAYBE
		}
	}
}

// updates the struct based on the resolution tag that's been mapped in the config file
func (f *FileInfo) getResolution() {
	cfg := config.Instance()
	k, v := matchMap(f.TunerLog, cfg.Local.Resolutions)
	f.Resolution.Tag = *k
	f.Resolution.Value = *v
}

func (f *FileInfo) getLength() {
	f.RecordedLength = -1
	for _, line := range f.TunerLog {
		if strings.Contains(line, ") Stop") {
			startIndex := strings.Index(line, "/")
			endIndex := strings.Index(line, "(")
			if startIndex != -1 && endIndex != -1 {
				slice := strings.Trim(line[startIndex+1:endIndex], " ")
				time := strings.Split(slice, ":")
				hours, _ := strconv.ParseInt(time[0], 10, 32)
				minutes, _ := strconv.ParseInt(time[1], 10, 32)
				f.RecordedLength = int(minutes + (hours * 60))
			}
		}
	}
	f.Length = -1
	for _, line := range f.MetadataLog {
		if strings.Contains(line, "Duration=") {
			slice := strings.Split(line, "=")[1]
			time := strings.Split(slice, ":")
			hours, _ := strconv.ParseInt(time[0], 10, 32)
			minutes, _ := strconv.ParseInt(time[1], 10, 32)
			f.Length = int(minutes + (hours * 60))
		}
	}
}

// Returns, in percent from 0-100, the difference in length between the recorded and actual length
func (f *FileInfo) LengthDifference() int {
	return int(math.Round(100 - (float64(f.Length) / float64(f.RecordedLength) * 100)))
}

// reads both log files and updates the struct
func (f *FileInfo) readLogs() error {
	stem := strings.TrimSuffix(f.Path, filepath.Ext(f.Path))
	tunerLogPath := stem + ".log"
	metadataLogPath := stem + ".txt"
	legacyLogPaths := []string{stem + ".mkv.log", stem + ".mpg.log"}

	if err := readFileContent(&f.MetadataLog, metadataLogPath); err != nil {
		_ = glg.Warnf("couldn't read metadata log: %s", err)
	} else {
		f.LogPaths = append(f.LogPaths, metadataLogPath)
	}
	if err := readFileContent(&f.TunerLog, tunerLogPath); err != nil {
		if err == os.ErrNotExist {
			for _, legacyLogPath := range legacyLogPaths {
				if err := readFileContent(&f.TunerLog, legacyLogPath); err == nil {
					_ = glg.Infof("legacy log file detected: %s", legacyLogPath)
					f.legacy = true
					f.LogPaths = append(f.LogPaths, legacyLogPath)
					return nil
				}
			}
		}
		return err
	}
	f.LogPaths = append(f.LogPaths, tunerLogPath)
	return nil
}

// Use to determine whether this file has a legacy logfile attached to it.
//
// If this teturns true, the MetadataLog will be nil as it doesn't exist for legacy file types
func (f *FileInfo) Legacy() bool {
	return f.legacy
}

func Find(slice []string, terms []string) bool {
	for _, line := range slice {
		for _, term := range terms {
			if strings.Contains(line, term) {
				return true
			}
		}
	}
	return false
}

func matchMap(slice []string, terms map[string]string) (*string, *string) {
	for _, line := range slice {
		for k, v := range terms {
			if strings.Contains(line, v) {
				return &k, &v
			}
		}
	}
	return nil, nil
}

func readFileContent(out *[]string, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	fileHandle, err := os.Open(filePath)
	if err != nil {
		_ = glg.Errorf("couldn't open log with path %s: %s", filePath, err)
		return err
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		*out = append(*out, fmt.Sprintln(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("error reading tuner log with path %s: %s", filePath, err)
		*out = nil
		return err
	}
	return nil
}
