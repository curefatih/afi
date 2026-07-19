package workers

import (
	"encoding/json"

	"github.com/curefatih/afi/internal/usage"
)

func EncodeUsage(e usage.Event) ([]byte, error) {
	return json.Marshal(e)
}
