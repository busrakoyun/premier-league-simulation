<script setup lang="ts">
import { useLeague } from '@/composables/useLeague'

const { predictions, predictionsAvailable, currentWeek } = useLeague()
</script>

<template>
  <section class="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
    <header class="border-b border-slate-200 bg-slate-50 px-4 py-2">
      <h2 class="text-sm font-semibold text-slate-700">
        {{ currentWeek > 0 ? `Week ${currentWeek} Championship Predictions` : 'Championship Predictions' }}
      </h2>
    </header>

    <div class="p-3">
      <p
        v-if="!predictionsAvailable"
        class="px-2 py-3 text-sm italic text-slate-500"
      >
        Predictions become available after the 4<sup>th</sup> week.
      </p>

      <ul v-else class="space-y-2">
        <li
          v-for="row in predictions"
          :key="row.team_id"
          class="flex items-center justify-between px-2 py-1"
        >
          <span class="text-sm text-slate-900">{{ row.team_name }}</span>
          <span class="text-sm font-semibold tabular-nums text-pl-purple-700">
            {{ Math.round(row.championship_rate * 100) }}%
          </span>
        </li>
      </ul>
    </div>
  </section>
</template>
