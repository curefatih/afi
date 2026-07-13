import { createRootRouteWithContext } from '@tanstack/react-router'


declare module '@tanstack/react-router' {
  interface StaticDataRouteOption {
    getTitle?: () => string
  }
}
