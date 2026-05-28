package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	httpapi "github.com/busrakoyun/premier-league-simulation/internal/http"
	"github.com/busrakoyun/premier-league-simulation/internal/prediction"
	"github.com/busrakoyun/premier-league-simulation/internal/service"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
	"github.com/busrakoyun/premier-league-simulation/internal/testutil"
)

func setupServer(t *testing.T) http.Handler {
	t.Helper()
	teams := []domain.Team{
		{ID: 1, Name: "Chelsea", ShortName: "CHE", AttackStrength: 1.30, DefenseStrength: 1.20, HomeAdvantage: 1.30},
		{ID: 2, Name: "Arsenal", ShortName: "ARS", AttackStrength: 1.05, DefenseStrength: 1.05, HomeAdvantage: 1.30},
		{ID: 3, Name: "Manchester City", ShortName: "MCI", AttackStrength: 1.00, DefenseStrength: 1.00, HomeAdvantage: 1.30},
		{ID: 4, Name: "Liverpool", ShortName: "LIV", AttackStrength: 0.85, DefenseStrength: 0.85, HomeAdvantage: 1.30},
	}
	repo := testutil.NewFakeLeagueRepo(teams)
	sim := simulation.NewPoissonSimulator()
	rng := simulation.NewRNG(42)
	pred := prediction.NewMonteCarlo(sim, 200, 42)
	svc := service.New(repo, sim, pred, rng)
	if err := svc.EnsureSeason(context.Background(), "test"); err != nil {
		t.Fatalf("EnsureSeason: %v", err)
	}
	return httpapi.NewHandler(svc).Router()
}

// ---- request helpers -------------------------------------------------------

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var br *strings.Reader
	if body != "" {
		br = strings.NewReader(body)
	}
	var req *http.Request
	if br != nil {
		req = httptest.NewRequest(method, path, br)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	return v
}

// ---- tests -----------------------------------------------------------------

func TestHealthz(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetTeams_ReturnsFourSeededTeams(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/api/seasons/current/teams", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.TeamsResponse](t, rec)
	if len(body.Teams) != 4 {
		t.Errorf("got %d teams, want 4", len(body.Teams))
	}
	if body.Teams[0].Name == "" || body.Teams[0].AttackStrength == 0 {
		t.Errorf("team payload looks empty: %+v", body.Teams[0])
	}
}

func TestGetCurrentSeason(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/api/seasons/current", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.SeasonDTO](t, rec)
	if body.TotalWeeks != 6 || body.CurrentWeek != 0 {
		t.Errorf("season DTO: %+v", body)
	}
}

func TestGetStandings_InitiallyZeroedRows(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/api/seasons/current/standings", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.StandingsResponse](t, rec)
	if len(body.Rows) != 4 {
		t.Fatalf("rows: got %d, want 4", len(body.Rows))
	}
	for _, r := range body.Rows {
		if r.Played != 0 || r.Points != 0 {
			t.Errorf("row should be zeroed: %+v", r)
		}
	}
}

func TestGetMatches_All12FixturesUnplayed(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/api/seasons/current/matches", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.MatchesResponse](t, rec)
	if len(body.Matches) != 12 {
		t.Errorf("matches: got %d, want 12", len(body.Matches))
	}
	for _, m := range body.Matches {
		if m.Played || m.Score != nil {
			t.Errorf("match %d should be unplayed: %+v", m.ID, m)
		}
		if m.HomeTeamName == "" || m.AwayTeamName == "" {
			t.Errorf("match %d missing team name: %+v", m.ID, m)
		}
	}
}

func TestGetPredictions_Returns422BeforeWeek4(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/api/seasons/current/predictions", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422, body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.ErrorResponse](t, rec)
	if body.Code != httpapi.CodePredictionsNotAvailable {
		t.Errorf("code=%q, want %q", body.Code, httpapi.CodePredictionsNotAvailable)
	}
}

func TestNextWeek_PlaysAndAdvances(t *testing.T) {
	h := setupServer(t)
	rec := do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.NextWeekResponse](t, rec)
	if body.Week != 1 {
		t.Errorf("Week=%d, want 1", body.Week)
	}
	if len(body.Matches) != 2 {
		t.Errorf("matches: got %d, want 2", len(body.Matches))
	}
	for _, m := range body.Matches {
		if !m.Played || m.Score == nil {
			t.Errorf("match %d should be played: %+v", m.ID, m)
		}
	}
}

func TestPredictions_AvailableAfter4Weeks(t *testing.T) {
	h := setupServer(t)
	for i := 0; i < 4; i++ {
		rec := do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("week %d next-week: status=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
	rec := do(t, h, http.MethodGet, "/api/seasons/current/predictions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("predictions status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.PredictionsResponse](t, rec)
	if len(body.Rows) != 4 {
		t.Errorf("rows: got %d, want 4", len(body.Rows))
	}
}

func TestPlayAll_FinishesSeasonAndReturnsByWeek(t *testing.T) {
	h := setupServer(t)
	rec := do(t, h, http.MethodPost, "/api/seasons/current/play-all", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.PlayAllResponse](t, rec)
	if len(body.Weeks) != 6 {
		t.Errorf("weeks: got %d, want 6", len(body.Weeks))
	}
	for i, w := range body.Weeks {
		if w.Week != i+1 {
			t.Errorf("weeks[%d].Week = %d, want %d", i, w.Week, i+1)
		}
		if len(w.Matches) != 2 {
			t.Errorf("week %d matches: got %d, want 2", w.Week, len(w.Matches))
		}
	}
}

func TestNextWeek_AfterSeasonComplete_Returns422(t *testing.T) {
	h := setupServer(t)
	// burn the season
	rec := do(t, h, http.MethodPost, "/api/seasons/current/play-all", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("play-all setup: %d %s", rec.Code, rec.Body.String())
	}
	rec = do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422 body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.ErrorResponse](t, rec)
	if body.Code != httpapi.CodeSeasonComplete {
		t.Errorf("code=%q, want %q", body.Code, httpapi.CodeSeasonComplete)
	}
}

func TestPatchMatch_EditPlayedMatchUpdatesStandings(t *testing.T) {
	h := setupServer(t)
	rec := do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("next-week: %d %s", rec.Code, rec.Body.String())
	}
	nw := decode[httpapi.NextWeekResponse](t, rec)
	m := nw.Matches[0]

	// 5-0 in favour of the home team.
	body := `{"home_goals":5,"away_goals":0}`
	rec = do(t, h, http.MethodPatch, "/api/matches/"+strconv.FormatInt(m.ID, 10), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	updated := decode[httpapi.MatchDTO](t, rec)
	if updated.Score == nil || updated.Score.HomeGoals != 5 || updated.Score.AwayGoals != 0 {
		t.Errorf("updated match score wrong: %+v", updated)
	}

	// Standings now reflect the edit.
	rec = do(t, h, http.MethodGet, "/api/seasons/current/standings", "")
	st := decode[httpapi.StandingsResponse](t, rec)
	var homeRow httpapi.StandingDTO
	for _, r := range st.Rows {
		if r.TeamID == m.HomeTeamID {
			homeRow = r
			break
		}
	}
	if homeRow.GoalsFor < 5 {
		t.Errorf("home team should have ≥5 GF after edit, got %+v", homeRow)
	}
}

func TestPatchMatch_RejectsBadID(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodPatch, "/api/matches/not-an-int", `{"home_goals":1,"away_goals":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400, body=%s", rec.Code, rec.Body.String())
	}
	body := decode[httpapi.ErrorResponse](t, rec)
	if body.Code != httpapi.CodeValidation {
		t.Errorf("code=%q want %q", body.Code, httpapi.CodeValidation)
	}
}

func TestPatchMatch_RejectsUnplayed_Returns422(t *testing.T) {
	h := setupServer(t)
	// Grab the first match (unplayed).
	rec := do(t, h, http.MethodGet, "/api/seasons/current/matches", "")
	body := decode[httpapi.MatchesResponse](t, rec)
	if len(body.Matches) == 0 {
		t.Fatal("no matches")
	}
	first := body.Matches[0]

	rec = do(t, h, http.MethodPatch, "/api/matches/"+strconv.FormatInt(first.ID, 10), `{"home_goals":1,"away_goals":0}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422 body=%s", rec.Code, rec.Body.String())
	}
	er := decode[httpapi.ErrorResponse](t, rec)
	if er.Code != httpapi.CodeMatchNotPlayed {
		t.Errorf("code=%q want %q", er.Code, httpapi.CodeMatchNotPlayed)
	}
}

func TestPostReset_ZerosWeekAndKeeps12Unplayed(t *testing.T) {
	h := setupServer(t)
	do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")
	do(t, h, http.MethodPost, "/api/seasons/current/next-week", "")

	rec := do(t, h, http.MethodPost, "/api/seasons/current/reset", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = do(t, h, http.MethodGet, "/api/seasons/current", "")
	season := decode[httpapi.SeasonDTO](t, rec)
	if season.CurrentWeek != 0 {
		t.Errorf("CurrentWeek after reset = %d, want 0", season.CurrentWeek)
	}

	rec = do(t, h, http.MethodGet, "/api/seasons/current/matches", "")
	matches := decode[httpapi.MatchesResponse](t, rec)
	if len(matches.Matches) != 12 {
		t.Errorf("matches after reset: got %d, want 12", len(matches.Matches))
	}
	for _, m := range matches.Matches {
		if m.Played {
			t.Errorf("match %d should be unplayed after reset", m.ID)
		}
	}
}

func TestSPA_ServesPlaceholderAtRoot(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("/ status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Premier League Simulation") {
		t.Errorf("placeholder content not found in / response")
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Content-Type=%q, want text/html*", rec.Header().Get("Content-Type"))
	}
}

func TestSPA_FallsBackToIndexForUnknownPath(t *testing.T) {
	rec := do(t, setupServer(t), http.MethodGet, "/some/client-side/route", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Premier League Simulation") {
		t.Errorf("SPA fallback should serve index.html")
	}
}
