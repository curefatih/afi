package quota

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(
	quotas []Quota,
	usage []Usage,
) Decision {

	for _, q := range quotas {

		if !q.Enabled {
			continue
		}

		for _, u := range usage {

			if u.Metric != q.Metric {
				continue
			}

			if q.Used.Add(u.Value).GreaterThan(q.Limit) {

				return Decision{
					Allowed:  false,
					Reason:   "quota exceeded",
					Violated: &q,
				}
			}
		}
	}

	return Decision{
		Allowed: true,
	}
}
