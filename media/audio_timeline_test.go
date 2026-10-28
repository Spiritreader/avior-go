package media

import (
	"testing"

	"github.com/Spiritreader/avior-go/config"
)

// The recorder log lists audio format changes with a timestamp, so the format
// that applies for the longest section is the recording's real audio format.
// This beats both "first tag wins" and ffprobe's 60 s window.
func TestDominantAudioFormat(t *testing.T) {
	tests := []struct {
		name string
		log  []string
		want AudioFormat
		ok   bool
	}{
		{
			// Production case: short stereo lead-in, then 5.1 for the film,
			// stereo again for the following programme.
			name: "Longlegs",
			log: []string{
				"22:13:12 / 00:00:01 (~ 1,27 MB) PID 6122: AC3 Audio Stereo, 48 khz, 448 kbps",
				"22:13:27 / 00:00:16 (~ 22,71 MB) PID 6122: AC3 Audio 5.1, 48 khz, 448 kbps",
				"23:47:08 / 01:33:57 (~ 6265,49 MB) PID 6122: AC3 Audio Stereo, 48 khz, 448 kbps",
				"23:47:37 / 01:34:26 (~ 6303,00 MB) Stop",
			},
			want: MULTI,
			ok:   true,
		},
		{
			// Only 3 seconds of 5.1, then stereo for the whole recording.
			name: "Bayern",
			log: []string{
				"02:40:31 / 00:00:01 (~ 0,64 MB) PID 5316: AC3 Audio 5.1, 48 khz, 384 kbps",
				"02:40:34 / 00:00:04 (~ 3,43 MB) PID 5316: AC3 Audio Stereo, 48 khz, 384 kbps",
				"03:14:35 / 00:34:04 (~ 1500,00 MB) Stop",
			},
			want: STEREO,
			ok:   true,
		},
		{
			name: "stereo only",
			log: []string{
				"20:00:01 / 00:00:01 (~ 1,00 MB) PID 6110: AC3 Audio Stereo, 48 khz, 448 kbps",
				"20:45:00 / 00:45:00 (~ 3000,00 MB) Stop",
			},
			want: STEREO,
			ok:   true,
		},
		{
			name: "no stop line falls back to last event",
			log: []string{
				"20:00:01 / 00:00:01 (~ 1,00 MB) PID 6110: AC3 Audio Stereo, 48 khz, 448 kbps",
				"20:00:20 / 00:00:20 (~ 20,00 MB) PID 6110: AC3 Audio 5.1, 48 khz, 448 kbps",
			},
			want: AUDIO_UNKNOWN,
			ok:   false,
		},
		{
			name: "no audio lines",
			log:  []string{"Timer Name: Irgendwas", "20:00:01 / 00:00:01 (~ 1,00 MB) Start Recording"},
			want: AUDIO_UNKNOWN,
			ok:   false,
		},
	}
	for _, tt := range tests {
		got, ok := dominantAudioFormat(tt.log)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("%s: dominantAudioFormat = (%s, %v), want (%s, %v)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

// End to end through getAudioFromLogs: the production Longlegs log must resolve
// to MULTI instead of the uncertain MULTI_PROBABLY/MULTI_MAYBE.
func TestAudioFromLogsUsesTimeline(t *testing.T) {
	cfg := config.Instance()
	prevFormats := cfg.Local.AudioFormats
	cfg.Local.AudioFormats = config.AudioFormats{
		StereoTags: []string{"AC3 Audio Stereo"},
		MultiTags:  []string{"AC3 Audio 5.1", "[5.1]"},
	}
	t.Cleanup(func() { cfg.Local.AudioFormats = prevFormats })

	longlegs := []string{
		"22:13:12 / 00:00:01 (~ 1,27 MB) PID 6110: H.264 Video, 16:9, 1280x720, 50 fps",
		"22:13:12 / 00:00:01 (~ 1,27 MB) PID 6122: AC3 Audio Stereo, 48 khz, 448 kbps",
		"22:13:27 / 00:00:16 (~ 22,71 MB) PID 6122: AC3 Audio 5.1, 48 khz, 448 kbps",
		"23:47:08 / 01:33:57 (~ 6265,49 MB) PID 6122: AC3 Audio Stereo, 48 khz, 448 kbps",
		"23:47:37 / 01:34:26 (~ 6303,00 MB) Stop",
	}
	f := File{TunerLog: longlegs}
	f.getAudioFromLogs()
	if f.AudioFormat != MULTI {
		t.Errorf("Longlegs audio = %s, want MULTI (5.1 covers ~93 min)", f.AudioFormat)
	}

	bayern := []string{
		"02:40:31 / 00:00:01 (~ 0,64 MB) PID 5316: AC3 Audio 5.1, 48 khz, 384 kbps",
		"02:40:34 / 00:00:04 (~ 3,43 MB) PID 5316: AC3 Audio Stereo, 48 khz, 384 kbps",
		"03:14:35 / 00:34:04 (~ 1500,00 MB) Stop",
	}
	f2 := File{TunerLog: bayern}
	f2.getAudioFromLogs()
	if f2.AudioFormat != STEREO {
		t.Errorf("Bayern audio = %s, want STEREO (stereo covers ~34 min)", f2.AudioFormat)
	}
}
