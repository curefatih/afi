package access

import (
	"fmt"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// KeyKind is a typed API key kind.
type KeyKind string

const (
	KeyKindPersonal       KeyKind = snapshot.KeyKindPersonal
	KeyKindServiceAccount KeyKind = snapshot.KeyKindServiceAccount
)

// ParseKeyKind validates and returns a KeyKind. Empty defaults to service_account.
func ParseKeyKind(kind string) (KeyKind, error) {
	kind = NormalizeKind(kind)
	switch KeyKind(kind) {
	case KeyKindPersonal, KeyKindServiceAccount:
		return KeyKind(kind), nil
	default:
		return "", fmt.Errorf("%w: invalid key kind %q", kernel.ErrInvalidRequest, kind)
	}
}

func (k KeyKind) String() string { return string(k) }

// IsPersonal reports whether the kind is personal.
func (k KeyKind) IsPersonal() bool { return k == KeyKindPersonal }
