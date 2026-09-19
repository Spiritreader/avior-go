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

// Produktionsfall 2026-09-19: "Mythos Marbella - Der Traum vom ewigen Sommer"
// hat KEIN Erscheinungsjahr in den Metadaten. Die Description nennt 1954 im
// Fließtext ("Den Grundstein legt 1954 Prinz Alfonso ...") — das ist
// Handlungszeit. Der Job wurde fälschlich zu "... (1954)" umbenannt und als
// Jahres-Kollision gegen die echte Version (2024) behandelt.
func TestRealMythosMarbellaNarrativeYearIgnored(t *testing.T) {
	f := File{
		Name: "Mythos Marbella - Der Traum vom ewigen Sommer",
		MetadataLog: []string{
			"[General]",
			"Version=1.1",
			"[Media]",
			"Created=15.09.2026 22:27:25",
			"Channel=3sat HD (AC3,deu)",
			"[0]",
			"Id=54281",
			"Date=15.09.2026",
			"Time=22:29:00",
			"Duration=00:44:00",
			"Title=Mythos Marbella - Der Traum vom ewigen Sommer",
			"Info=Film von Hannes Schuler",
			"Description=Vom Geheimtipp der High Society zum touristischen Hotspot Europas: Marbella steht wie kaum eine andere Stadt für Glamour, Reichtum und internationale Prominenz.||Den Grundstein legt 1954 Prinz Alfonso zu Hohenlohe mit der Eröffnung des \"Marbella Club\". Adel, Superreiche und Hollywoodstars entdecken die malerische Kulisse am Mittelmeer und machen Marbella zum Treffpunkt des internationalen Jetsets.|HD-Produktion||[16:9]   [PDC 15.09. 22:25]",
			"Charset=255",
			"Content=144",
			"MinimumAge=0",
		},
		TunerLog: []string{
			"3sat HD (AC3,deu) 15/09/2026",
			`\\192.168.178.75\recording_pool\recording\Mythos Marbella - Der Traum vom ewigen Sommer_2026-09-15-22-27-01-3sat HD (AC3,deu).ts`,
			"Naming Scheme: %event_%year-%date-%time-%station",
			"Device: Tvheadend:9983 20d48b009f 5",
			"EventID: 54281, PDC: 0x7CD99",
			"Timer Name: Mythos Marbella - Der Traum vom ewigen Sommer - Film von Hannes Schuler",
			"Timer Start: 15/09/2026 22:27:00",
			"Timer Duration: 00:48:00 (48 min. incl. 2 min. lead time, 2 min. follow-up time)",
			"Timer Options: Teletext=0, Subtitles=0, All Audio Tracks=0, Adjust PAT/PMT=1, EIT EPG Data=0, Transponder Dump=0",
			"Timer Source: Search:Mythos",
			"Monitoring Mode: Start/stop by running status",
			"22:27:25 / 00:00:00 (~ 0,51 MB) PID 6522: AC3 Audio Stereo, 48 khz, 448 kbps",
			"22:27:26 / 00:00:01 (~ 2,23 MB) PID 6510: H.264 Video, 16:9, 1280x720, 50 fps",
			"23:10:45 / 00:43:20 (~ 4311,93 MB) Stop",
		},
	}
	if y := f.ExtractYearFromFile(); y != "" {
		t.Errorf("Mythos Marbella: got %q, want \"\" (1954 ist Handlungsjahr, kein Erscheinungsjahr)", y)
	}
}
