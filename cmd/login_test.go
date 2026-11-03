package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/FacileStudio/porte/devicelogin"
)

func TestDeviceOutcome(t *testing.T) {
	if fallback, err := deviceOutcome(devicelogin.ErrExchangeNotMounted, "https://j"); !fallback || err != nil {
		t.Fatalf("not mounted = %v, %v, want a fallback", fallback, err)
	}
	fallback, err := deviceOutcome(devicelogin.ErrAccountNotLinked, "https://j")
	if fallback || err == nil || !strings.Contains(err.Error(), "sign in once at https://j") {
		t.Fatalf("not linked = %v, %v, want the sign-in hint", fallback, err)
	}
	if fallback, err := deviceOutcome(devicelogin.ErrDenied, "https://j"); fallback || !errors.Is(err, devicelogin.ErrDenied) {
		t.Fatalf("denied = %v, %v, want the error unchanged", fallback, err)
	}
	if fallback, err := deviceOutcome(nil, "https://j"); fallback || err != nil {
		t.Fatalf("success = %v, %v, want no fallback and no error", fallback, err)
	}
}
