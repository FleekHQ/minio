package s3x

import "testing"

func Test_convertToHashV0(t *testing.T) {
	type args struct {
		hash string
	}

	newHash := "bafybeighlvmeuez4gvsfl2pxad3pd7zvjlfxur3ryhigwad2ixw6gxhe3y"
	oldHash := "QmbktPvf9gX4AB6DB12dnuXQM5uDLWWTTKjk2ZhghWtWdT"

	tests := []struct {
		name string
		args args
		want string
	}{
		{"Success turn Hash V1 to V0", args{newHash}, "QmbktPvf9gX4AB6DB12dnuXQM5uDLWWTTKjk2ZhghWtWdT"},
		{"Bad Hash", args{"bad hash"}, ""},
		{"Handle V0 Hash", args{oldHash}, oldHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToHashV0(tt.args.hash); got != tt.want {
				t.Errorf("convertToHashV0() = %v, want %v", got, tt.want)
			}
		})
	}
}
