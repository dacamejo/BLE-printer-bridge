package ble

import "testing"

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "already normalized", input: "66:22:B6:5C:5C:3C", want: "66:22:B6:5C:5C:3C"},
		{name: "dashes and lowercase", input: "66-22-b6-5c-5c-3c", want: "66:22:B6:5C:5C:3C"},
		{name: "trims whitespace", input: " 66:22:B6:5C:5C:3C ", want: "66:22:B6:5C:5C:3C"},
		{name: "invalid group count", input: "66:22:B6:5C:3C", wantErr: true},
		{name: "invalid hex", input: "66:22:B6:5C:5C:ZZ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeAddress(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
