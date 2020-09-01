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
	Legacy         bool
}

func (f *FileInfo) Update() error {
	if !f.Legacy {
		if err := f.readLogs(); err != nil {
			return err
		}
		f.getAudio()
		f.getResolution()
	}
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
	keys := make([]string, 0, len(cfg.Local.Resolutions))
	for k := range cfg.Local.Resolutions {
		keys = append(keys, k)
	}
	matchedKey := match(f.TunerLog, keys)
	if matchedKey != nil {
		f.Resolution = cfg.Local.Resolutions[*matchedKey]
	}
}

// reads both log files and updates the struct
func (f *FileInfo) readLogs() error {
	tunerLogPath := strings.TrimSuffix(f.Path, filepath.Ext(f.Path)) + ".log"
	metadataLogPath := strings.TrimSuffix(f.Path, filepath.Ext(f.Path)) + ".txt"
	tunerFile, err := os.Open(tunerLogPath)
	if err != nil {
		_ = glg.Errorf("couldn't open tuner log: %s", err)
		return err
	}
	defer tunerFile.Close()
	metadataFile, err := os.Open(metadataLogPath)
	if err != nil {
		_ = glg.Errorf("couldn't open metadata log: %s", err)
		return err
	}
	defer metadataFile.Close()

	scanner := bufio.NewScanner(tunerFile)
	for scanner.Scan() {
		f.TunerLog = append(f.TunerLog, fmt.Sprintln(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("error reading tuner log: %s", err)
		f.TunerLog = nil
		return err
	}

	scanner = bufio.NewScanner(metadataFile)
	for scanner.Scan() {
		f.MetadataLog = append(f.MetadataLog, fmt.Sprintln(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		_ = glg.Errorf("error reading metadata log: %s", err)
		f.TunerLog = nil
		f.MetadataLog = nil
		return err
	}
	return nil
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

func match(slice []string, terms []string) *string {
	for _, line := range slice {
		for _, term := range terms {
			if strings.Contains(line, term) {
				return &term
			}
		}
	}
	return nil
}
