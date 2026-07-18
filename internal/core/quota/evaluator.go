package quota

import "github.com/curefatih/afi/internal/core/usage"

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e Evaluator) Evaluate(
	quotas []Quota,
	use []usage.Usage,
) Decision {

	index := make(map[usage.Metric]Quota)

	for _, q := range quotas {
		if q.Enabled {
			index[q.Metric] = q
		}
	}

	for _, u := range use {

		q, ok := index[u.Metric]
		if !ok {
			continue
		}

		if q.Used.Add(u.Value).GreaterThan(q.Limit) {
			return Decision{
				Allowed: false,
				Metric:  u.Metric,
			}
		}
	}

	return Decision{
		Allowed: true,
	}
}
