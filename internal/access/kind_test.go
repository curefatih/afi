package access

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestParseKeyKind(t *testing.T) {
	t.Parallel()
	k, err := ParseKeyKind("")
	if err != nil || k != KeyKindServiceAccount {
		t.Fatalf("got %q err=%v", k, err)
	}
	k, err = ParseKeyKind(snapshot.KeyKindPersonal)
	if err != nil || !k.IsPersonal() {
		t.Fatalf("got %q err=%v", k, err)
	}
	if _, err := ParseKeyKind("nope"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
