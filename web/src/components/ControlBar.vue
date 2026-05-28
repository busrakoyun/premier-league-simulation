<script setup lang="ts">
import { useLeague } from '@/composables/useLeague'

const { nextWeek, playAll, reset, loading, seasonFinished } = useLeague()
</script>

<template>
  <div
    class="flex items-center justify-between rounded-lg border border-slate-200 bg-white p-3 shadow-sm"
  >
    <button
      type="button"
      class="rounded px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="loading || seasonFinished"
      @click="playAll"
    >
      Play All
    </button>

    <button
      type="button"
      class="text-sm font-medium text-slate-500 transition-colors hover:text-slate-800 disabled:opacity-50"
      :disabled="loading"
      @click="reset"
    >
      Reset season
    </button>

    <button
      type="button"
      class="rounded bg-pl-purple-700 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-pl-purple-800 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="loading || seasonFinished"
      @click="nextWeek"
    >
      Next Week
    </button>
  </div>
</template>
