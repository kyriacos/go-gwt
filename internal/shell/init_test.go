package shell

import (
	"strings"
	"testing"
)

const testBin = "/opt/homebrew/bin/gwt"

func TestInitSupportedShells(t *testing.T) {
	for _, sh := range Shells() {
		t.Run(sh, func(t *testing.T) {
			script, err := Init(sh, "gwt", testBin)
			if err != nil {
				t.Fatalf("Init(%q) returned error: %v", sh, err)
			}
			if strings.TrimSpace(script) == "" {
				t.Fatalf("Init(%q) returned empty script", sh)
			}
			if !strings.Contains(script, testBin) {
				t.Errorf("Init(%q) script does not pin binary path %q", sh, testBin)
			}
			if !strings.Contains(script, "gwt") {
				t.Errorf("Init(%q) script does not mention the gwt function", sh)
			}
			if !strings.Contains(script, "GWT_PATH_OUT") {
				t.Errorf("Init(%q) script does not use GWT_PATH_OUT cd protocol", sh)
			}
			for _, verb := range []string{"new", "from", "co", "checkout", "search", "pick"} {
				if !strings.Contains(script, verb) {
					t.Errorf("Init(%q) script missing switch verb %q", sh, verb)
				}
			}
			if !strings.Contains(script, "--help") {
				t.Errorf("Init(%q) script should pass through --help", sh)
			}
		})
	}
}

func TestInitDefinesFunction(t *testing.T) {
	cases := map[string]string{
		"zsh":  "gwt() {",
		"bash": "gwt() {",
		"fish": "function gwt",
	}
	for sh, want := range cases {
		script, err := Init(sh, "gwt", testBin)
		if err != nil {
			t.Fatalf("Init(%q): %v", sh, err)
		}
		if !strings.Contains(script, want) {
			t.Errorf("Init(%q) script missing function definition %q", sh, want)
		}
	}
}

func TestInitUnknownShell(t *testing.T) {
	if _, err := Init("powershell", "gwt", testBin); err == nil {
		t.Fatal("Init(\"powershell\") expected an error, got nil")
	}
	if _, err := Init("", "gwt", testBin); err == nil {
		t.Fatal("Init(\"\") expected an error, got nil")
	}
}

func TestInitCustomName(t *testing.T) {
	wantFn := map[string]string{"zsh": "oldgwt() {", "bash": "oldgwt() {", "fish": "function oldgwt"}
	for _, sh := range Shells() {
		script, err := Init(sh, "oldgwt", testBin)
		if err != nil {
			t.Fatalf("Init(%q, oldgwt): %v", sh, err)
		}
		if !strings.Contains(script, wantFn[sh]) {
			t.Errorf("Init(%q, oldgwt) missing renamed function %q", sh, wantFn[sh])
		}
		if !strings.Contains(script, testBin) {
			t.Errorf("Init(%q, oldgwt) does not pin binary path", sh)
		}
		if !strings.Contains(script, "GWT_POPULATE:") {
			t.Errorf("Init(%q, oldgwt) mangled GWT_POPULATE sentinel", sh)
		}
	}
}

func TestShells(t *testing.T) {
	got := Shells()
	want := []string{"zsh", "bash", "fish"}
	if len(got) != len(want) {
		t.Fatalf("Shells() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Shells() = %v, want %v", got, want)
		}
	}
}
