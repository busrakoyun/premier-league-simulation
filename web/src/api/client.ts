import type {
  ApiError,
  Match,
  MatchesResponse,
  NextWeekResponse,
  PlayAllResponse,
  PredictionsResponse,
  Season,
  StandingsResponse,
  TeamsResponse,
} from '@/types/api'

// ApiResponseError is the typed exception thrown for any non-2xx response.
// Components catch it to display the human message and branch on `code`
// for behaviour (e.g., hide the predictions panel when the server returns
// PREDICTIONS_NOT_YET_AVAILABLE).
export class ApiResponseError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiResponseError'
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(init.headers ?? {}),
    },
    ...init,
  })

  if (res.status === 204) {
    return undefined as T
  }

  const body = (await res.json().catch(() => null)) as unknown
  if (!res.ok) {
    const err = body as ApiError | null
    throw new ApiResponseError(
      res.status,
      err?.code ?? 'UNKNOWN',
      err?.error ?? `HTTP ${res.status}`,
    )
  }
  return body as T
}

// All endpoints exposed by the Go backend. One function per route keeps the
// surface readable; the underlying request() handles JSON + error envelope.
export const api = {
  getCurrentSeason: (): Promise<Season> => request('/api/seasons/current'),
  getTeams: (): Promise<TeamsResponse> => request('/api/seasons/current/teams'),
  getStandings: (): Promise<StandingsResponse> =>
    request('/api/seasons/current/standings'),
  getMatches: (): Promise<MatchesResponse> =>
    request('/api/seasons/current/matches'),
  getPredictions: (): Promise<PredictionsResponse> =>
    request('/api/seasons/current/predictions'),
  postNextWeek: (): Promise<NextWeekResponse> =>
    request('/api/seasons/current/next-week', { method: 'POST' }),
  postPlayAll: (): Promise<PlayAllResponse> =>
    request('/api/seasons/current/play-all', { method: 'POST' }),
  postReset: (): Promise<void> =>
    request('/api/seasons/current/reset', { method: 'POST' }),
  patchMatch: (id: number, home_goals: number, away_goals: number): Promise<Match> =>
    request(`/api/matches/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ home_goals, away_goals }),
    }),
}
