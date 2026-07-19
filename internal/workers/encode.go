package workers

import "encoding/json"

func EncodeUsage(e UsagePayload) ([]byte, error) {
	return json.Marshal(e)
}
