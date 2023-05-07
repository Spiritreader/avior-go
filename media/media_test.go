package media

import (
	"fmt"
	"testing"

	"github.com/Spiritreader/avior-go/consts"
)

func TestSanitize (t *testing.T) {
	//testFile := &File{Path: "D:\\Recording\\Monaco 110 - Madonna di Napoli.mkv"}
	//testFile := &File{Path: "D:\\Recording\\Neva Give üp - Der einzig wahre Japaner.mkv"}
	testFile := &File{Path: "D:\\Temp\\test.log.log"}
	testFile.Update()
	testFile.SanitizeLog()
	contains, _ := testFile.LogsContain([]string{"-rc vbr_hq -qmin 16 -qmax 23"}, []string{consts.MODULE_NAME_LOGMATCH})
	fmt.Printf("contains line: %t\n", contains)
	contains, _ = testFile.LogsContain([]string{"-c:v hevc_nvenc -preset p7 -tune hq"}, []string{consts.MODULE_NAME_LOGMATCH})
	fmt.Printf("contains line: %t\n", contains)
}
