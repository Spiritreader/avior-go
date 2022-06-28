package media

import "testing"

func TestSanitize (t *testing.T) {
	//testFile := &File{Path: "D:\\Recording\\Monaco 110 - Madonna di Napoli.mkv"}
	testFile := &File{Path: "D:\\Recording\\Neva Give üp - Der einzig wahre Japaner.mkv"}
	testFile.Update()
	testFile.SanitizeLog()
}
