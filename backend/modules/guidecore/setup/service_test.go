package setup

import "testing"

func TestNeedSetupFromShadowComparesRootEntries(t *testing.T) {
	rom := []byte("daemon:*:0:0:99999:7:::\nroot:$1$abc:0:0:99999:7:::\n")

	tests := []struct {
		name    string
		current []byte
		want    bool
	}{
		{
			name:    "root entry unchanged",
			current: []byte("bin:*:0:0:99999:7:::\nroot:$1$abc:0:0:99999:7:::\n"),
			want:    true,
		},
		{
			name:    "root entry changed",
			current: []byte("root:$1$changed:0:0:99999:7:::\n"),
			want:    false,
		},
		{
			name:    "missing root entry",
			current: []byte("daemon:*:0:0:99999:7:::\n"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedSetupFromShadow(rom, tt.current)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestBuildNeedSetupInfo(t *testing.T) {
	tests := []struct {
		name string
		in   NeedSetupInput
		need bool
		wifi bool
	}{
		{
			name: "show guide disabled",
			in: NeedSetupInput{
				ShowGuide:         false,
				PasswordCheckOK:   true,
				PasswordUnchanged: true,
				HasWireless:       true,
			},
			need: false,
			wifi: true,
		},
		{
			name: "already setup",
			in: NeedSetupInput{
				ShowGuide:         true,
				SetupMarked:       true,
				PasswordCheckOK:   true,
				PasswordUnchanged: true,
			},
			need: false,
		},
		{
			name: "root password unchanged",
			in: NeedSetupInput{
				ShowGuide:         true,
				PasswordCheckOK:   true,
				PasswordUnchanged: true,
			},
			need: true,
		},
		{
			name: "password check unavailable",
			in: NeedSetupInput{
				ShowGuide:         true,
				PasswordCheckOK:   false,
				PasswordUnchanged: true,
			},
			need: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNeedSetupInfo(tt.in)
			if got.Need != tt.need {
				t.Fatalf("expected need %v, got %v", tt.need, got.Need)
			}
			if got.Wifi != tt.wifi {
				t.Fatalf("expected wifi %v, got %v", tt.wifi, got.Wifi)
			}
		})
	}
}
