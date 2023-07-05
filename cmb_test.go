package main

import (
	"fmt"
	"testing"
)

func Test_emailParseCmb(t *testing.T) {
	type args struct {
		cfg fetchConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				cfg: fetchConfig{
					Username: "",
					Password: "",
					Hour:     300,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := emailParseCmb(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("emailParseCmb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(got)
		})
	}
}
