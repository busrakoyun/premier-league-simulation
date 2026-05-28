<script setup lang="ts">
import { useLeague } from '@/composables/useLeague'

const { standings } = useLeague()

function signed(n: number): string {
  return n > 0 ? `+${n}` : `${n}`
}
</script>

<template>
  <section class="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
    <header class="border-b border-slate-200 bg-slate-50 px-4 py-2">
      <h2 class="text-sm font-semibold text-slate-700">League Table</h2>
    </header>
    <table class="w-full text-sm">
      <thead class="bg-slate-50 text-xs uppercase text-slate-500">
        <tr>
          <th class="px-3 py-2 text-left font-medium">Team</th>
          <th class="px-2 py-2 text-center font-medium">PTS</th>
          <th class="px-2 py-2 text-center font-medium">P</th>
          <th class="px-2 py-2 text-center font-medium">W</th>
          <th class="px-2 py-2 text-center font-medium">D</th>
          <th class="px-2 py-2 text-center font-medium">L</th>
          <th class="px-2 py-2 text-center font-medium">GD</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, i) in standings"
          :key="row.team_id"
          :class="[
            'border-t border-slate-100',
            i === 0 && row.points > 0 ? 'bg-pl-purple-50' : '',
          ]"
        >
          <td class="px-3 py-2 font-medium text-slate-900">{{ row.team_name }}</td>
          <td class="px-2 py-2 text-center font-semibold tabular-nums">{{ row.points }}</td>
          <td class="px-2 py-2 text-center tabular-nums text-slate-600">{{ row.played }}</td>
          <td class="px-2 py-2 text-center tabular-nums text-slate-600">{{ row.won }}</td>
          <td class="px-2 py-2 text-center tabular-nums text-slate-600">{{ row.drawn }}</td>
          <td class="px-2 py-2 text-center tabular-nums text-slate-600">{{ row.lost }}</td>
          <td class="px-2 py-2 text-center tabular-nums text-slate-600">
            {{ signed(row.goal_difference) }}
          </td>
        </tr>
        <tr v-if="standings.length === 0">
          <td colspan="7" class="px-3 py-4 text-center text-sm italic text-slate-500">
            Loading…
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
