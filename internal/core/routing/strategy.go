package routing

type Strategy string

const (
	StrategyFirstHealthy Strategy = "FIRST_HEALTHY"

	StrategyRoundRobin Strategy = "ROUND_ROBIN"

	StrategyWeighted Strategy = "WEIGHTED"

	StrategyRandom Strategy = "RANDOM"

	StrategyFailover Strategy = "FAILOVER"
)