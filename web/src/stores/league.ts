import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { ApiResponseError, api } from '@/api/client'
import type {
  Match,
  PredictionRow,
  Season,
  StandingRow,
  Team,
} from '@/types/api'

// PREDICTIONS_VISIBLE_FROM_WEEK mirrors the backend constant. Hard-coded
// here because the contract is fixed by the PDF (predictions appear from
// the 4th week onward); not worth a runtime fetch.
const PREDICTIONS_VISIBLE_FROM_WEEK = 4

export const useLeagueStore = defineStore('league', () => {
  // ---- Raw state -----------------------------------------------------------
  const season = ref<Season | null>(null)
  const teams = ref<Team[]>([])
  const matches = ref<Match[]>([])
  const standings = ref<StandingRow[]>([])
  const predictions = ref<PredictionRow[]>([])

  // ---- UI state ------------------------------------------------------------
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ---- Derived state -------------------------------------------------------
  const currentWeek = computed(() => season.value?.current_week ?? 0)
  const totalWeeks = computed(() => season.value?.total_weeks ?? 0)

  const currentWeekMatches = computed<Match[]>(() => {
    if (currentWeek.value === 0) return []
    return matches.value.filter((m) => m.week === currentWeek.value)
  })

  const predictionsAvailable = computed(
    () => currentWeek.value >= PREDICTIONS_VISIBLE_FROM_WEEK,
  )

  const seasonFinished = computed(
    () => totalWeeks.value > 0 && currentWeek.value >= totalWeeks.value,
  )

  // ---- Error helper --------------------------------------------------------
  function setError(e: unknown) {
    if (e instanceof ApiResponseError) {
      error.value = e.message
    } else if (e instanceof Error) {
      error.value = e.message
    } else {
      error.value = String(e)
    }
  }

  function clearError() {
    error.value = null
  }

  // ---- Actions -------------------------------------------------------------

  // fetchAll is the single source of truth for what's on screen. Every
  // mutating action calls it after committing — keeps the UI consistent
  // with the server without optimistic state machines.
  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      const [s, t, m, st] = await Promise.all([
        api.getCurrentSeason(),
        api.getTeams(),
        api.getMatches(),
        api.getStandings(),
      ])
      season.value = s
      teams.value = t.teams
      matches.value = m.matches
      standings.value = st.rows

      if (s.current_week >= PREDICTIONS_VISIBLE_FROM_WEEK) {
        const p = await api.getPredictions()
        predictions.value = p.rows
      } else {
        predictions.value = []
      }
    } catch (e) {
      setError(e)
    } finally {
      loading.value = false
    }
  }

  async function nextWeek() {
    loading.value = true
    error.value = null
    try {
      await api.postNextWeek()
      await fetchAll()
    } catch (e) {
      setError(e)
      loading.value = false
    }
  }

  async function playAll() {
    loading.value = true
    error.value = null
    try {
      await api.postPlayAll()
      await fetchAll()
    } catch (e) {
      setError(e)
      loading.value = false
    }
  }

  async function reset() {
    loading.value = true
    error.value = null
    try {
      await api.postReset()
      await fetchAll()
    } catch (e) {
      setError(e)
      loading.value = false
    }
  }

  async function editMatch(matchID: number, homeGoals: number, awayGoals: number) {
    loading.value = true
    error.value = null
    try {
      await api.patchMatch(matchID, homeGoals, awayGoals)
      await fetchAll()
    } catch (e) {
      setError(e)
      loading.value = false
    }
  }

  return {
    // state
    season,
    teams,
    matches,
    standings,
    predictions,
    loading,
    error,
    // derived
    currentWeek,
    totalWeeks,
    currentWeekMatches,
    predictionsAvailable,
    seasonFinished,
    // actions
    fetchAll,
    nextWeek,
    playAll,
    reset,
    editMatch,
    clearError,
  }
})
