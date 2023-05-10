package encoder

import (
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/media"
)

func TestEncode(t *testing.T) {
	config.LoadLocalFrom("../config.json")
	testFile := media.File{Path: `\\UMS\recording_pool\Manual\Thomas Hengelbrock dirigiert Ravel und Franck.mkv`}
	testFile.Update()
	dst := "D:\\Recording\\testencode"
	Encode(testFile, 0, 0, false, &dst)
}
