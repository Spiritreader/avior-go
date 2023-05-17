package tools

import "testing"

func TestDurationVerify(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "fail test",
			path: "D:\\Recording\\2075 - Verbrannte Erde.mkv",
			want: false,
		},
		{
			name: "success test",
			path: "D:\\Recording\\Kofelgschroa_2018-05-02-22-43-01-BR Süd HD (AC3,deu).ts",
			want: true,
		},
		{
			name: "success test",
			path: "D:\\Recording\\testencode\\Ein Fall Für Zwei - Verhängnisvolle Freundschaft.mkv",
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got, err := FfProbeDurationVerify(test.path); got != test.want {
				if err != nil {
					t.Errorf("DurationVerify() = %v, want %v, error: %v", got, test.want, err)
				} else {
					t.Errorf("DurationVerify() = %v, want %v", got, test.want)
				}
			} else {
				t.Logf("DurationVerify() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestByteCountUpIEC(t *testing.T) {
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
			name: "test",
			args: args{
				b:      100000000000,
				outExp: 4,
			},
			want:  0.09094947017729282,
			want1: "0.1 TiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ByteCountUpSI(tt.args.b, tt.args.outExp)
			if got != tt.want {
				t.Errorf("ByteCountUpIEC() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ByteCountUpIEC() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
