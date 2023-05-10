package media

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
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
	AudioFormat      AudioFormat
	Errors           int
	CustomParams     []string
	MetadataLog      []string
	TunerLog         []string
	LogPaths         []string
	AllowReplacement bool
	legacy           bool
}

// Updates the struct to fill out all remaining fields
func (f *File) Update() error {
	f.RecordedLength = -1
	f.Length = -1
	if err := f.readLogs(); err != nil {
		return err
	}
	f.getAudio()
	f.getResolution()
	f.getLength()
	f.getErrors()
	f.trimName()
	found, _, idx := find(f.CustomParams, []string{consts.MODULE_FLAG_SKIP, "lengthOverride"}, nil)
	if found {
		f.AllowReplacement = true
		f.CustomParams = append(f.CustomParams[:idx], f.CustomParams[idx+1:]...)
	}
	return nil
}

// LogsContain returns true once the first term matches.
//
// It also includes the term that was matched against
func (f *File) LogsContain(terms []string, ignoredLines []string) (bool, string) {
	tunerContains, tMatch, _ := find(f.TunerLog, terms, ignoredLines)
	if tunerContains {
		return true, tMatch
	}
	metadataContains, mMatch, _ := find(f.MetadataLog, terms, ignoredLines)
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

func (f *File) SanitizeLog() error {
	found, term, idx := find(f.TunerLog, []string{consts.LOG_DELIM, "VDRAvior:"}, nil)
	save := false
	if found {
		if (idx - 1) < 0 {
			idx = 0
		} else {
			idx = idx - 1
		}
		f.TunerLog = f.TunerLog[:idx]
		_ = glg.Infof("removing previous statistics from log (term: %s)", term)
		save = true
	} else {
		found, _, idx := find(f.TunerLog, []string{"OriginalPath: "}, nil)
		if found {
			if (idx - 2) < 0 {
				idx = 0
			} else {
				idx = idx - 2
			}
			f.TunerLog = f.TunerLog[:idx]
			_ = glg.Infof("removing previous statistics from log without header")
			save = true
		}
	}

	if save {
		file, err := os.OpenFile(f.LogPaths[0], os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			_ = glg.Errorf("could not sanitize tuner log file for %s, error: %s", f.LogPaths[0], err)
			_ = glg.Errorf("dumping tuner log file contents: %+v", f.TunerLog)
			return err
		}
		defer file.Close()
		for _, line := range f.TunerLog {
			_, err = file.WriteString(line)
			if err != nil {
				_ = glg.Errorf("failed while sanitizing tuner log file for %s, error: %s", f.LogPaths[0], err)
				_ = glg.Errorf("dumping tuner log file contents: %+v", f.TunerLog)
				return err
			}
		}
	}
	return nil
}

// getAudio retrieves the audio file from the log files and updates the struct
func (f *File) getAudio() {

	f.getAudioFromLogs()

	if f.AudioFormat == AUDIO_UNKNOWN {
		channels, channel_layout, err := tools.FfProbeChannels(f.Path)
		if err != nil {
			return
		}
		glg.Logf("ffprobe reports %d channels with layout %s for %s", channels, channel_layout, f.Path)
		if channels == 2 || channel_layout == "stereo" {
			f.AudioFormat = STEREO
		}
		if channels > 2 || channel_layout == "5.1(side)" || channel_layout == "5.1" {
			f.AudioFormat = MULTI
		}
	}
}

func (f *File) getAudioFromLogs() {
	cfg := config.Instance()

	tunerStereo, _, _ := find(f.TunerLog, cfg.Local.AudioFormats.StereoTags, nil)
	tunerMulti, _, _ := find(f.TunerLog, cfg.Local.AudioFormats.MultiTags, nil)

	if tunerStereo && !tunerMulti {
		// guaranteed to be stereo if tuner only picks up one audio codec
		f.AudioFormat = STEREO
	} else if !tunerStereo && tunerMulti {
		// guaranteed to be multichannel if tuner only picks up one audio codec
		f.AudioFormat = MULTI
	} else if tunerStereo && tunerMulti {
		// complement info with tags if available
		metaStereo, _, _ := find(f.MetadataLog, cfg.Local.AudioFormats.StereoTags, nil)
		metaMulti, _, _ := find(f.MetadataLog, cfg.Local.AudioFormats.MultiTags, nil)

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

// retrieves the recorded length and the expected length from the metadata
func (f *File) getLength() {
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

// collect the amount of errors found in the recording
func (f *File) getErrors() {
	f.Errors = -1
	// disable legacy error counting for now, needs more investigation
	if f.legacy {
		return
	}
	errors := findAllLines(f.TunerLog, []string{"Errors:"})
	errorCount := 0
	for _, line := range errors {
		countString := strings.Split(line[strings.Index(line, "Errors:"):], ":")[1]
		count, err := strconv.ParseInt(strings.Trim(countString, " \n"), 10, 32)
		if err == nil {
			errorCount += int(count)
		}
	}
	f.Errors = errorCount
}

// Removes unwanted strings from the Output file name
func (f *File) trimName() {
	cfg := config.Instance()
	terms := findAllTerms([]string{f.Name}, cfg.Shared.NameExclude)
	sort.Slice(terms, func(i, j int) bool {
		return len(terms[i]) > len(terms[j])
	})
	for _, term := range terms {
		if idx := strings.Index(f.Name, term); idx != -1 {
			f.Name = strings.Trim(f.Name[idx+len(term):], " ")
		}
	}
	terms = findAllTerms([]string{f.Subtitle}, cfg.Shared.SubExclude)
	sort.Slice(terms, func(i, j int) bool {
		return len(terms[i]) > len(terms[j])
	})
	trimPerformed := false
	for _, term := range terms {
		if idx := strings.Index(f.Subtitle, term); idx != -1 {
			trimPerformed = true
			f.Subtitle = strings.Trim(f.Subtitle[:idx], " ")
		}
	}
	if trimPerformed && strings.HasSuffix(f.Subtitle, "-") {
		f.Subtitle = f.Subtitle[:len(f.Subtitle)-1]
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
		var pe *os.PathError
		if err == os.ErrNotExist || errors.As(err, &pe) {
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
	// mitigate empty files
	if len(f.TunerLog) <= 2 {
		_ = glg.Warnf("tuner log seems to be empty, marking file as legacy")
		f.legacy = true
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

// find checks whether any string in terms is found in a slice.
//
// find returns once the first match is found.
//
// It returns three values. a boolean indicating whether a term occurred,
// the term it found and its index within the slice
func find(slice []string, terms []string, ignoredLines []string) (bool, string, int) {
	for idx, line := range slice {
		if ignoredLines != nil {
			skip := false
			for _, ignored := range ignoredLines {
				if strings.HasPrefix(line, ignored) {
					skip = true
				}
			}
			if skip {
				continue
			}
		}
		for _, term := range terms {
			if len(term) > 0 && strings.Contains(line, term) {
				return true, term, idx
			}
		}
	}
	return false, "", -1
}

// Returns all lines in the slice that match one of the terms in the terms slice
//
// Returns all terms that have been found
func findAllTerms(slice []string, terms []string) []string {
	found := make([]string, 0)
	for _, line := range slice {
		for _, term := range terms {
			if len(term) > 0 && strings.Contains(line, term) {
				found = append(found, term)
			}
		}
	}
	return found
}

// Returns all lines in the slice that match one of the terms in the terms slice
//
// Returns all lines where a match occurred
func findAllLines(slice []string, terms []string) []string {
	found := make([]string, 0)
	for _, line := range slice {
		for _, term := range terms {
			if len(term) > 0 && strings.Contains(line, term) {
				found = append(found, line)
			}
		}
	}
	return found
}

// Checks if the value of a map is present within any line within the slice.
//
// Returns the key and value if found
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
