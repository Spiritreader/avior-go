package media

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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

type FileInfo struct {
	Path           string
	Name           string
	Subtitle       string
	Resolution     string
	RecordedLength string
	Length         string
	AudioFormat    int
	MetadataLog    []string
	TunerLog       []string
	legacy         bool
}

func (f *FileInfo) Update() error {
	if err := f.readLogs(); err != nil {
		return err
	}
	f.getAudio()
	f.getResolution()
	return nil
}

// getAudio retrieves the audio file from the log files and updates the struct
func (f *FileInfo) getAudio() {
	cfg := config.Instance()

	tunerStereo := find(f.TunerLog, cfg.Local.AudioFormats.StereoTags)
	tunerMulti := find(f.TunerLog, cfg.Local.AudioFormats.MultiTags)

	if tunerStereo && !tunerMulti {
		// guaranteed to be stereo if tuner only picks up one audio codec
		f.AudioFormat = STEREO
	} else if !tunerStereo && tunerMulti {
		// guaranteed to be multichannel if tuner only picks up one audio codec
		f.AudioFormat = MULTI
	} else if tunerStereo && tunerMulti {
		// complement info with tags if available
		metaStereo := find(f.MetadataLog, cfg.Local.AudioFormats.StereoTags)
		metaMulti := find(f.MetadataLog, cfg.Local.AudioFormats.MultiTags)

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
	f.Resolution = *match(f.TunerLog, cfg.Local.Resolutions)
}

// reads both log files and updates the struct
func (f *FileInfo) readLogs() error {
	stem := strings.TrimSuffix(f.Path, filepath.Ext(f.Path))
	tunerLogPath := stem + ".log"
	metadataLogPath := stem + ".txt"
	legacyLogPaths := []string{stem + ".mkv.log", stem + ".mpg.log"}

	if err := readFileContent(&f.MetadataLog, metadataLogPath); err != nil {
		_ = glg.Warnf("couldn't read metadata log: %s", err)
	}
	if err := readFileContent(&f.TunerLog, tunerLogPath); err != nil {
		if err == os.ErrNotExist {
			for _, legacyLogPath := range legacyLogPaths {
				if err := readFileContent(&f.TunerLog, legacyLogPath); err == nil {
					_ = glg.Infof("legacy log file detected: %s", legacyLogPath)
					f.legacy = true
					return nil
				}
			}
		}
		return err
	}
	return nil
}

// Use to determine whether this file has a legacy logfile attached to it.
//
// If this teturns true, the MetadataLog will be nil as it doesn't exist for legacy file types
func (f *FileInfo) Legacy() bool {
	return f.legacy
}

func find(slice []string, terms []string) bool {
	for _, line := range slice {
		for _, term := range terms {
			if strings.Contains(line, term) {
				return true
			}
		}
	}
	return false
}

func match(slice []string, terms map[string]string) *string {
	for _, line := range slice {
		for k, v := range terms {
			if strings.Contains(line, v) {
				return &k
			}
		}
	}
	return nil
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
