package plugin

type Hook string

const (
	HookAuthenticate Hook = "AUTHENTICATE"

	HookBeforeRoute Hook = "BEFORE_ROUTE"

	HookAfterRoute Hook = "AFTER_ROUTE"

	HookBeforeProvider Hook = "BEFORE_PROVIDER"

	HookAfterProvider Hook = "AFTER_PROVIDER"

	HookUsage Hook = "USAGE"

	HookPricing Hook = "PRICING"

	HookCompleted Hook = "COMPLETED"

	HookError Hook = "ERROR"
)
