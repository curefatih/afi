declare module "@tanstack/react-router" {
	interface StaticDataRouteOption {
		getTitle?: () => string;
	}
}
