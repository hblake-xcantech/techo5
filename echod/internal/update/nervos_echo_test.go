//go:build nervos_echo && !dot && !spot && !panel

package update

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNervOSReleaseIdentity(t *testing.T) {
	if !Published() || Stable.URL() != "https://github.com/hblake-xcantech/techo5/releases/latest/download/manifest.json" {
		t.Fatalf("fork stable origin: %q", Stable.URL())
	}
	if !strings.HasSuffix(Dev.URL(), "/download/dev/manifest.json") {
		t.Fatal("dev channel changed")
	}
	if releaseKey == "KVUuQUbhyKwPBbIneqFEvXYSI+3Hkfu/heCTy5YNVMk=" {
		t.Fatal("fork trusts upstream signing key")
	}
	key, err := base64.StdEncoding.DecodeString(releaseKey)
	if err != nil || len(key) != 32 {
		t.Fatal("fork signing key is invalid")
	}
}
