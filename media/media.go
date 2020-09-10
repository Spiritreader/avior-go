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
	"github.com/Spiritreader/avior-go/tools"
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

type AudioFormat int

const (
	STEREO          = AudioFormat(-3)
	STEREO_PROBABLY = AudioFormat(-2)
	STEREO_MAYBE    = AudioFormat(-1)
	AUDIO_UNKNOWN   = AudioFormat(0)
	MULTI_MAYBE     = AudioFormat(1)
	MULTI_PROBABLY  = AudioFormat(2)
	MULTI           = AudioFormat(3)
)

func (a AudioFormat) String() string {
	switch a {
	case STEREO:
		return "STEREO"
	case STEREO_PROBABLY:
		return "STEREO_PROBABLY"
	case STEREO_MAYBE:
		return "STEREO_MAYBE"
	case MULTI_PROBABLY:
		return "MULTI_PROBABLY"
	case MULTI_MAYBE:
		return "MULTI_MAYBE"
	case MULTI:
		return "MULTI"
	}
	return "UNKNOWN"
}

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

type File struct {
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
	AudioFormat  AudioFormat
	CustomParams []string
	MetadataLog  []string
	TunerLog     []string
	LogPaths     []string
	IgnoreLength bool
	legacy       bool
}

// Updates the struct to fill out all remaining fields
func (f *File) Update() error {
	if err := f.readLogs(); err != nil {
		return err
	}
	f.getAudio()
	f.getResolution()
	f.getLength()
	f.trimName()
	found, _, idx := find(f.CustomParams, []string{"lengthOverride"})
	if found {
		f.IgnoreLength = true
		f.CustomParams = append(f.CustomParams[:idx], f.CustomParams[idx+1:]...)
	}
	return nil
}

// LogsContain returns true once the first term matches.
//
// It also includes the term that was matched against
func (f *File) LogsContain(terms []string) (bool, string) {
	tunerContains, tMatch, _ := find(f.TunerLog, terms)
	if tunerContains {
		return true, tMatch
	}
	metadataContains, mMatch, _ := find(f.MetadataLog, terms)
	if metadataContains {
		return true, mMatch
	}
	return false, ""
}

// Returns, in percent from 0-100, the difference in length between the recorded and actual length
func (f *File) LengthDifference() int {
	return int(math.Round(100 - (float64(f.RecordedLength) / float64(f.Length) * 100)))
}

func (f *File) OutName() string {
	if len(f.Subtitle) == 0 {
		return f.Name
	}
	return f.Name + " - " + f.Subtitle
}

// getAudio retrieves the audio file from the log files and updates the struct
func (f *File) getAudio() {
	cfg := config.Instance()

	tunerStereo, _, _ := find(f.TunerLog, cfg.Local.AudioFormats.StereoTags)
	tunerMulti, _, _ := find(f.TunerLog, cfg.Local.AudioFormats.MultiTags)

	if tunerStereo && !tunerMulti {
		// guaranteed to be stereo if tuner only picks up one audio codec
		f.AudioFormat = STEREO
	} else if !tunerStereo && tunerMulti {
		// guaranteed to be multichannel if tuner only picks up one audio codec
		f.AudioFormat = MULTI
	} else if tunerStereo && tunerMulti {
		// complement info with tags if available
		metaStereo, _, _ := find(f.MetadataLog, cfg.Local.AudioFormats.StereoTags)
		metaMulti, _, _ := find(f.MetadataLog, cfg.Local.AudioFormats.MultiTags)

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
func (f *File) getResolution() {
	cfg := config.Instance()
	k, v := matchMap(f.TunerLog, cfg.Local.Resolutions)
	if k != nil && v != nil {
		f.Resolution.Tag = *k
		f.Resolution.Value = *v
	}
}

func (f *File) getLength() {
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

// Removes unwanted strings from the Output file name
func (f *File) trimName() {
	cfg := config.Instance()
	terms := findAll([]string{f.Name}, cfg.Shared.NameExclude)
	for _, term := range terms {
		if idx := strings.Index(f.Name, term); idx != -1 {
			f.Name = strings.Trim(f.Name[idx+len(term):], " ")
		}
	}
	terms = findAll([]string{f.Subtitle}, cfg.Shared.SubExclude)
	for _, term := range terms {
		if idx := strings.Index(f.Subtitle, term); idx != -1 {
			f.Subtitle = strings.Trim(f.Subtitle[:idx], " ")
		}
	}
	f.Name = strings.Trim(tools.RemoveIllegalChars(f.Name), " ")
	f.Subtitle = strings.Trim(tools.RemoveIllegalChars(f.Subtitle), " ")
}

// reads both log files and updates the struct
func (f *File) readLogs() error {
	stem := strings.TrimSuffix(f.Path, filepath.Ext(f.Path))
	tunerLogPath := stem + ".log"
	metadataLogPath := stem + ".txt"
	legacyLogPaths := []string{stem + ".mkv.log", stem + ".mpg.log"}

	mErr := readFileContent(&f.MetadataLog, metadataLogPath)
	if err := readFileContent(&f.TunerLog, tunerLogPath); err != nil {
		if err == os.ErrNotExist {
			for _, legacyLogPath := range legacyLogPaths {
				if err := readFileContent(&f.TunerLog, legacyLogPath); err == nil {
					_ = glg.Logf("legacy log file detected: %s", legacyLogPath)
					f.legacy = true
					f.LogPaths = append(f.LogPaths, legacyLogPath)
					return nil
				}
			}
		}
		return err
	}
	f.LogPaths = append(f.LogPaths, tunerLogPath)

	if mErr != nil {
		_ = glg.Warnf("could not read metadata log for \"%s\": %s", metadataLogPath, mErr)
	} else {
		f.LogPaths = append(f.LogPaths, metadataLogPath)
	}
	return nil
}

// Use to determine whether this file has a legacy logfile attached to it.
//
// If this teturns true, the MetadataLog will be nil as it doesn't exist for legacy file types
func (f *File) Legacy() bool {
	return f.legacy
}

func find(slice []string, terms []string) (bool, string, int) {
	for idx, line := range slice {
		for _, term := range terms {
			if len(term) > 0 && strings.Contains(line, term) {
				return true, term, idx
			}
		}
	}
	return false, "", -1
}

func findAll(slice []string, terms []string) []string {
	found := make([]string, 0)
	for _, line := range slice {
		for _, term := range terms {
			if strings.Contains(line, term) {
				found = append(found, term)
			}
		}
	}
	return found
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
		_ = glg.Errorf("could not open log with path \"%s\": %s", filePath, err)
		return err
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		*out = append(*out, fmt.Sprintln(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("error reading tuner log with path \"%s\": %s", filePath, err)
		*out = nil
		return err
	}
	return nil
}
