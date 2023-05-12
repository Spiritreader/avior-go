package joblog

import (
	"testing"
)

func TestEncode(t *testing.T) {
	var jobLog Data
	jobLog.Add("testoooo")
	jobLog2 := jobLog
	jobLog2.Add("test2")
	jobLog3 := &jobLog
	t.Logf("%s", jobLog.messages)
	t.Logf("%s", jobLog2.messages)
	t.Logf("%s", jobLog3.messages)
}
