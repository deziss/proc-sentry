package internal

import (
	"os"
	"testing"
)

func TestLoadUserMap(t *testing.T) {
	content := "root:x:0:0:root:/root:/bin/bash\nnobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin\nanshu:x:1000:1000:Anshu:/home/anshu:/bin/bash\n"

	f, err := os.CreateTemp("", "passwd-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	m := LoadUserMap(f.Name())

	if len(m) != 3 {
		t.Errorf("expected 3 users, got %d", len(m))
	}
	if m["0"] != "root" {
		t.Errorf("expected uid 0 = root, got %q", m["0"])
	}
	if m["1000"] != "anshu" {
		t.Errorf("expected uid 1000 = anshu, got %q", m["1000"])
	}
	if m["65534"] != "nobody" {
		t.Errorf("expected uid 65534 = nobody, got %q", m["65534"])
	}
}

func TestLoadUserMapMissing(t *testing.T) {
	m := LoadUserMap("/nonexistent/passwd")
	if len(m) != 0 {
		t.Errorf("expected empty map for missing file, got %d entries", len(m))
	}
}

func TestResolveUser(t *testing.T) {
	m := map[string]string{"0": "root", "1000": "anshu"}

	if got := ResolveUser(m, "0"); got != "root" {
		t.Errorf("expected root, got %q", got)
	}
	if got := ResolveUser(m, "1000"); got != "anshu" {
		t.Errorf("expected anshu, got %q", got)
	}
	if got := ResolveUser(m, "9999"); got != "9999" {
		t.Errorf("expected fallback 9999, got %q", got)
	}
}
