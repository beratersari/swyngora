package domain

import "testing"

func TestNormalizeClientID(t *testing.T) {
	ok, err := NormalizeClientID("  user-1_abc.2  ")
	if err != nil || ok != "user-1_abc.2" {
		t.Fatalf("got %q err=%v", ok, err)
	}
	for _, bad := range []string{"", "default", "DEFAULT", "anonymous", "http-default", "ai-assistant", "tg-12345", "tg-1", "has space", "bad!"} {
		if _, err := NormalizeClientID(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	// UUID-style telegram mapping ids remain valid.
	if _, err := NormalizeClientID("tg-not-digits"); err != nil {
		t.Fatalf("non-numeric tg- suffix should pass charset check: %v", err)
	}
	long := stringsRepeat("a", MaxClientIDLen+1)
	if _, err := NormalizeClientID(long); err == nil {
		t.Fatal("expected too long")
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
