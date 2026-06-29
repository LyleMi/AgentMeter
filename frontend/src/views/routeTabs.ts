import { computed } from 'vue'
import { useRoute } from 'vue-router'

export interface RouteTabMatch {
  key: string
  pathPrefix: string
}

export function routeTabKey(path: string, matches: readonly RouteTabMatch[], fallback: string) {
  return matches.find((item) => path.startsWith(item.pathPrefix))?.key || fallback
}

export function useRouteTabKey(matches: readonly RouteTabMatch[], fallback: string) {
  const route = useRoute()
  return computed(() => routeTabKey(route.path, matches, fallback))
}
