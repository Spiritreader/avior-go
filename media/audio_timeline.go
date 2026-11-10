package media

import (
	"regexp"
	"strconv"
	"strings"
)

// audioEventRe captures the elapsed seconds and the audio description of a
// recorder log line:
//
//	22:13:12 / 00:00:16 (~ 22,71 MB) PID 6122: AC3 Audio 5.1, 48 khz, 448 kbps
//	           ^^^^^^^^ elapsed
var audioEventRe = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\s*/\s*(\d{2}):(\d{2}):(\d{2})\s*\([^)]*\)\s*(.*)$`)

// stopEventRe matches the recording stop line of a recorder log.
var stopEventRe = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\s*/\s*(\d{2}):(\d{2}):(\d{2})\s*\([^)]*\)\s*Stop\b`)

// audioFormatInLineRe is the recorder's own wording for the audio PID line.
var audioFormatInLineRe = regexp.MustCompile(`(?i)PID\s+\d+:\s*(.+)$`)

// dominantAudioFormat derives the recording's real audio format from the
// recorder log timeline: every format change starts a section, and the format
// covering the most seconds wins.
//
// The log states when a format applies, which is strictly more information than
// a tag search ("both tags present") or ffprobe's fixed look-ahead window: a
// film whose PID switches to 5.1 after a short stereo lead-in is 5.1, while a
// recording with three seconds of 5.1 and 34 minutes of stereo is stereo.
//
// Returns the format and true only when a stop line plus at least one format
// event exist; otherwise the caller keeps its previous (tag based) logic.
func dominantAudioFormat(lines []string) (AudioFormat, bool) {
	type event struct {
		at  int
		fmt AudioFormat
	}
	events := make([]event, 0, 4)
	stopAt, hasStop := 0, false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := stopEventRe.FindStringSubmatch(line); len(m) == 4 {
			stopAt, hasStop = hmsToSeconds(m[1], m[2], m[3]), true
			break
		}
		m := audioEventRe.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		at := hmsToSeconds(m[1], m[2], m[3])
		pid := audioFormatInLineRe.FindStringSubmatch(m[4])
		if len(pid) != 2 {
			continue
		}
		switch detectAudioFormat(pid[1]) {
		case MULTI:
			events = append(events, event{at, MULTI})
		case STEREO:
			events = append(events, event{at, STEREO})
		}
	}

	if !hasStop || len(events) == 0 {
		return AUDIO_UNKNOWN, false
	}

	durations := map[AudioFormat]int{}
	for i, ev := range events {
		end := stopAt
		if i+1 < len(events) {
			end = events[i+1].at
		}
		if end > ev.at {
			durations[ev.fmt] += end - ev.at
		}
	}
	if len(durations) == 0 {
		return AUDIO_UNKNOWN, false
	}

	best := MULTI
	for _, fmt := range []AudioFormat{MULTI, STEREO} {
		if durations[fmt] > durations[best] {
			best = fmt
		}
	}
	if durations[best] == 0 {
		return AUDIO_UNKNOWN, false
	}
	return best, true
}

// detectAudioFormat maps a recorder audio description to the format family.
// Multichannel wording is checked first so "AC3 Audio 5.1" never counts as
// stereo because of the shared "AC3 Audio" prefix.
func detectAudioFormat(desc string) AudioFormat {
	switch {
	case strings.Contains(desc, "5.1") || strings.Contains(desc, "5.0"),
		strings.Contains(strings.ToLower(desc), "multi"):
		return MULTI
	case strings.Contains(strings.ToLower(desc), "stereo"):
		return STEREO
	}
	return AUDIO_UNKNOWN
}

func hmsToSeconds(h, m, s string) int {
	hh, _ := strconv.Atoi(h)
	mm, _ := strconv.Atoi(m)
	ss, _ := strconv.Atoi(s)
	return hh*3600 + mm*60 + ss
}
