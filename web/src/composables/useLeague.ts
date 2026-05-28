import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'

import { useLeagueStore } from '@/stores/league'

// useLeague is the one composable every component touches. Wrapping the
// Pinia store this way (a) auto-fetches on mount so individual components
// don't all reinvent it, (b) destructures via storeToRefs so reactivity
// survives, and (c) gives us a single place to add cross-cutting concerns
// (polling, optimistic updates) later without touching components.
export function useLeague() {
  const store = useLeagueStore()
  const refs = storeToRefs(store)

  onMounted(() => {
    // Only fetch once per app lifetime; the store actions refetch as needed.
    if (refs.season.value === null && !refs.loading.value) {
      store.fetchAll()
    }
  })

  return {
    ...refs,
    nextWeek: store.nextWeek,
    playAll: store.playAll,
    reset: store.reset,
    editMatch: store.editMatch,
    fetchAll: store.fetchAll,
    clearError: store.clearError,
  }
}
