// Package hook is the stable lifecycle API for in-process gateway extensions.
//
// Register implementations on the dataplane HookChain from cmd/gateway:
//
//	hooks := dataplane.NewHookChain().
//	    RegisterHook(demohook.New()).
//	    RegisterBeforeCall(myBeforeCall)
//
// BeforeCall receives a mutable CallContext (principal, route, tags, metadata, body)
// and may allow, enrich, or deny. AfterCall runs after the provider attempt.
package hook
