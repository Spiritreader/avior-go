package tools

import (
	"testing"
)

func TestDurationVerify(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "fail test",
			path: "\\\\UMS\\media\\transcoded\\failed\\Abenteuer Wildnis - Wilde Dynastien - Königin der Löwen.mkv",
			want: false,
		},
		{
			name: "success test",
			path: "\\\\UMS\\media\\transcoded\\HD720\\Der Kommissar und die Alpen - Gute Gesellschaft.mkv",
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got, err := FfProbeVerfiy(test.path); got != test.want {
				if err != nil {
					t.Errorf("DurationVerify() = %v, want %v, error: %v", got, test.want, err)
				} else {
					t.Errorf("DurationVerify() = %v, want %v", got, test.want)
				}
			} else {
				if err != nil {
					t.Logf("DurationVerify() = %v, want %v, error: %v", got, test.want, err)
				} else {
					t.Logf("DurationVerify() = %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestByteCountUpSI(t *testing.T) {
	type args struct {
		b      int64
		outExp int
	}
	tests := []struct {
		name  string
		args  args
		want  float64
		want1 string
	}{
		{
			name: "terabytes",
			args: args{
				b:      100000000000,
				outExp: 4,
			},
			want:  0.1,
			want1: "0.1 TB",
		},
		{
			name: "gigabytes",
			args: args{
				b:      5000000000,
				outExp: 3,
			},
			want:  5,
			want1: "5.0 GB",
		},
		{
			name: "megabytes",
			args: args{
				b:      2500000,
				outExp: 2,
			},
			want:  2.5,
			want1: "2.5 MB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ByteCountUpSI(tt.args.b, tt.args.outExp)
			if got != tt.want {
				t.Errorf("ByteCountUpSI() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ByteCountUpSI() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
