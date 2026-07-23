package platform

import "context"

type actorCtxKey struct{}

// WithActor attaches the acting platform user id to ctx for domain event emission.
func WithActor(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return context.WithValue(ctx, actorCtxKey{}, userID)
}

// ActorFrom returns the acting user id previously set with WithActor, or "".
func ActorFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(actorCtxKey{}).(string)
	return v
}
