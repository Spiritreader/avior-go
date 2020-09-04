package tools

import "testing"

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
				b: 100000000000,
				outExp: 4,
			},
			want: 0.09094947017729282,
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
