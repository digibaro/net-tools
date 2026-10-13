package main

import "testing"

func TestHealthURL(t *testing.T) {
	tests := []struct {
		listen, want string
		wantErr      bool
	}{
		{"0.0.0.0:8081", "http://127.0.0.1:8081/api/healthz", false},
		{":8081", "http://127.0.0.1:8081/api/healthz", false},
		{"[::]:8081", "http://127.0.0.1:8081/api/healthz", false},
		{"127.0.0.1:9000", "http://127.0.0.1:9000/api/healthz", false},
		{"[::1]:8081", "http://[::1]:8081/api/healthz", false},
		{"8081", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.listen, func(t *testing.T) {
			got, err := healthURL(tt.listen)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
