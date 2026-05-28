<script setup lang="ts">
import LeagueTable from '@/components/LeagueTable.vue'
import MatchResults from '@/components/MatchResults.vue'
import PredictionsPanel from '@/components/PredictionsPanel.vue'
import ControlBar from '@/components/ControlBar.vue'
import { useLeague } from '@/composables/useLeague'

const { error, clearError, season, seasonFinished } = useLeague()
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto max-w-6xl px-4 py-4 flex items-center justify-between">
        <h1 class="text-lg font-semibold text-slate-900">Premier League Simulation</h1>
        <p v-if="season" class="text-sm text-slate-500">
          {{ seasonFinished ? 'Season finished' : `Week ${season.current_week} of ${season.total_weeks}` }}
        </p>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-4 py-6 space-y-4">
      <div
        v-if="error"
        class="flex items-center justify-between rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800"
      >
        <span>{{ error }}</span>
        <button
          class="ml-3 text-xs text-red-600 hover:text-red-800"
          @click="clearError"
        >
          Dismiss
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <LeagueTable />
        <MatchResults />
        <PredictionsPanel />
      </div>

      <ControlBar />
    </main>
  </div>
</template>
