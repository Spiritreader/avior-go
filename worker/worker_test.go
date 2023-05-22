package worker

import (
	"testing"

	"github.com/Spiritreader/avior-go/media"
)

func TestTraverse(t *testing.T) {
	file := media.File{
		Path: "\\\\UMS\\media\\transcoded\\2075 - Verbrannte Erde.mkv",
		Name: "2075 - Verbrannte Erde",
	}
	traverseDir(&file, "\\\\UMS\\media\\transcoded", false)
}
