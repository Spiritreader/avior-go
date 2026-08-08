package media

import "testing"

// Reale Fälle aus der Session — neuer Film mit .log (Timer Name 2024, Start 2026)
func TestRealNewFilmLog(t *testing.T) {
	f := File{
		Name: "Die Löwin",
		TunerLog: []string{
			"arte HD (deu) 03/08/2026",
			`\\192.168.178.75\recording_pool\recording\Die Löwin_2026-08-03-22-58-00-arte HD (deu).ts`,
			"Naming Scheme: %event_%year-%date-%time-%station",
			"Device: Tvheadend:9983 20d48b009f 3",
			"EventID: 63174, PDC: 0x1C5C0",
			"Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024",
			"Timer Start: 03/08/2026 22:58:00",
			"Timer Duration: 01:44:00 (104 min. incl. 2 min. lead time, 2 min. follow-up time)",
			"Timer Options: Teletext=0, Subtitles=0, All Audio Tracks=0, Adjust PAT/PMT=1, EIT EPG Data=0, Transponder Dump=0",
			"Timer Source: Search:Regex Fernsehfilm|Spielfilm|Liebesfilm|Thriller|Liebes",
			"Monitoring Mode: Start/stop by running status",
			"22:58:00 / 00:00:00 (~ 0,00 MB) Start EPG Monitoring",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("neuer Film .log: got %q, want 2024 (Timer Name)", y)
	}
}

// Neuer Film mit .txt (Info= 2024) — der eigentliche Produktionsfall
func TestRealNewFilmTxt(t *testing.T) {
	f := File{
		Name: "Die Löwin",
		MetadataLog: []string{
			"[Media]",
			"Created=03.08.2026 23:00:06",
			"Channel=arte HD (deu)",
			"[0]",
			"Id=63174",
			"Date=03.08.2026",
			"Title=Die Löwin",
			"Info=Spielfilm Deutschland/Estland/Lettland 2024",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("neuer Film .txt: got %q, want 2024 (Info=)", y)
	}
}

// Alter Film: nur .log, "Melodram Südafrika/2011"
func TestRealOldFilmLog(t *testing.T) {
	f := File{
		Name: "Die Löwin",
		TunerLog: []string{
			"ZDF HD 02.01.2012",
			"Die Löwin",
			"20:15..21:45",
			"Melodram Südafrika/2011",
			"20:10:02 Start",
			"Total Size 9078,8 MB",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2011" {
		t.Errorf("alter Film .log: got %q, want 2011", y)
	}
}

// Der Produktionsfall: Job bringt den Subtitle mit dem Release-Jahr 2024 mit,
// die .log des Films enthält aber auch das Aufnahmejahr 2026. Der Subtitle
// muss gewinnen (wird zuerst geprüft).
func TestRealSubtitleWinsOverLogRecordingYear(t *testing.T) {
	f := File{
		Name:     "Die Löwin",
		Subtitle: "Spielfilm Deutschland/Estland/Lettland 2024",
		TunerLog: []string{
			"arte HD (deu) 03/08/2026",
			"Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024",
			"Timer Start: 03/08/2026 22:58:00",
			"Monitoring Mode: Start/stop by running status",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("Subtitle-Pfad: got %q, want 2024 (Subtitle gewinnt vor .log 2026)", y)
	}
}
