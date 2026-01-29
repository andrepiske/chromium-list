package main

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		filename string
		want     chromiumVersion
		wantErr  bool
	}{
		{
			filename: "chromium_144.0.7559.96-1~deb13u1_amd64.deb",
			want: chromiumVersion{
				major:    144,
				minor:    0,
				build:    7559,
				patch:    96,
				revision: 1,
			},
			wantErr: false,
		},
		{
			filename: "chromium_143.0.7499.169-1~deb12u1_arm64.deb",
			want: chromiumVersion{
				major:    143,
				minor:    0,
				build:    7499,
				patch:    169,
				revision: 1,
			},
			wantErr: false,
		},
		{
			filename: "invalid_format.deb",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		got, err := parseVersion(tt.filename)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseVersion(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if got.major != tt.want.major || got.minor != tt.want.minor { // Check basic fields
				t.Errorf("parseVersion(%q) = %+v, want %+v", tt.filename, got, tt.want)
			}
		}
	}
}

func TestCompareVersions(t *testing.T) {
	v1 := chromiumVersion{major: 144, minor: 0, build: 0, patch: 0}
	v2 := chromiumVersion{major: 143, minor: 0, build: 0, patch: 0}

	if !compareVersions(v1, v2) {
		t.Error("144 should be greater than 143")
	}

	v3 := chromiumVersion{major: 144, minor: 1, build: 0, patch: 0}
	if !compareVersions(v3, v1) {
		t.Error("144.1 should be greater than 144.0")
	}
}

func TestParseLinks(t *testing.T) {
	html := `<html><body>
	<a href="chromium_1.0.0.0_amd64.deb">link1</a>
	<a href="other_file.deb">link2</a>
	</body></html>`

	links, err := parseLinks(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseLinks failed: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
	if links[0] != "chromium_1.0.0.0_amd64.deb" {
		t.Errorf("unexpected link content: %s", links[0])
	}
}
