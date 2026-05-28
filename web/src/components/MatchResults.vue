<script setup lang="ts">
import { ref } from 'vue'

import { useLeague } from '@/composables/useLeague'
import type { Match } from '@/types/api'

const { currentWeekMatches, currentWeek, editMatch, loading } = useLeague()

// editing.matchID === null means "no row in edit mode". Switching rows
// auto-cancels the previous edit (only one inline form at a time).
const editing = ref<{ matchID: number | null; home: number; away: number }>({
  matchID: null,
  home: 0,
  away: 0,
})

function startEdit(match: Match) {
  if (!match.played || !match.score) return
  editing.value = {
    matchID: match.id,
    home: match.score.home_goals,
    away: match.score.away_goals,
  }
}

function cancelEdit() {
  editing.value = { matchID: null, home: 0, away: 0 }
}

async function saveEdit() {
  if (editing.value.matchID === null) return
  const { matchID, home, away } = editing.value
  cancelEdit()
  await editMatch(matchID, home, away)
}
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white shadow-sm">
    <header class="border-b border-slate-200 bg-slate-50 px-4 py-2">
      <h2 class="text-sm font-semibold text-slate-700">
        {{ currentWeek > 0 ? `Week ${currentWeek} Match Results` : 'Match Results' }}
      </h2>
    </header>

    <div class="space-y-1 p-3">
      <p
        v-if="currentWeekMatches.length === 0"
        class="px-2 py-3 text-sm italic text-slate-500"
      >
        No matches played yet. Click <span class="font-semibold">Next Week</span> to begin.
      </p>

      <div
        v-for="match in currentWeekMatches"
        :key="match.id"
        class="group flex items-center justify-between rounded px-2 py-2 transition-colors hover:bg-slate-50"
        :class="{ 'cursor-pointer': match.played && editing.matchID !== match.id }"
        @click="startEdit(match)"
      >
        <span class="flex-1 text-right text-sm text-slate-900">{{ match.home_team_name }}</span>

        <template v-if="editing.matchID !== match.id">
          <span class="mx-3 font-semibold tabular-nums text-slate-900">
            {{ match.score?.home_goals ?? '-' }} - {{ match.score?.away_goals ?? '-' }}
          </span>
        </template>

        <span
          v-else
          class="mx-2 flex items-center gap-1"
          @click.stop
        >
          <input
            v-model.number="editing.home"
            type="number"
            min="0"
            max="20"
            class="w-12 rounded border border-slate-300 px-1 py-0.5 text-center tabular-nums focus:border-pl-purple-600 focus:outline-none"
          />
          <span class="text-slate-400">-</span>
          <input
            v-model.number="editing.away"
            type="number"
            min="0"
            max="20"
            class="w-12 rounded border border-slate-300 px-1 py-0.5 text-center tabular-nums focus:border-pl-purple-600 focus:outline-none"
          />
          <button
            class="ml-2 text-xs font-semibold text-pl-purple-700 hover:text-pl-purple-800 disabled:opacity-50"
            :disabled="loading"
            @click.stop="saveEdit"
          >
            Save
          </button>
          <button
            class="ml-1 text-xs text-slate-500 hover:text-slate-700"
            @click.stop="cancelEdit"
          >
            Cancel
          </button>
        </span>

        <span class="flex-1 text-left text-sm text-slate-900">{{ match.away_team_name }}</span>
      </div>
    </div>
  </section>
</template>
