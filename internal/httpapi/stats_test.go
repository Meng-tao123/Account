package httpapi

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"personal-ledger/internal/ledger"
)

func statsTestRecords() []ledger.Record {
	return []ledger.Record{
		{Date: "2026-11-07", Type: "income", AmountCents: 300000, Category: "工资"},
		{Date: "2026-11-07", Type: "expense", AmountCents: 1250, Category: "饮食"},
		{Date: "2026-11-07", Type: "expense", AmountCents: 1750, Category: "交通"},
	}
}

func TestStatsFilters(t *testing.T) {
	tests := []struct {
		name, query string
		want        statsResult
	}{
		{"all", "", statsResult{300000, 3000, 297000}},
		{"category", "?category=交通", statsResult{0, 1750, -1750}},
		{"income category", "?category=工资", statsResult{300000, 0, 300000}},
		{"inclusive amounts", "?minCents=1250&maxCents=1750", statsResult{0, 3000, -3000}},
		{"exact amount", "?minCents=1250&maxCents=1250", statsResult{0, 1250, -1250}},
		{"inclusive dates", "?from=2026-11-07&to=2026-11-07", statsResult{300000, 3000, 297000}},
		{"no matches", "?from=2026-11-08", statsResult{}},
		{"to only", "?to=2026-11-06", statsResult{}},
		{"minimum only", "?minCents=1750", statsResult{300000, 1750, 298250}},
		{"maximum only", "?maxCents=1250", statsResult{0, 1250, -1250}},
		{"zero boundary", "?minCents=0", statsResult{300000, 3000, 297000}},
		{"maximum zero", "?maxCents=0", statsResult{}},
		{"empty values", "?from=&to=&category=&minCents=&maxCents=", statsResult{300000, 3000, 297000}},
		{"combined AND", "?from=2026-11-07&to=2026-11-07&category=交通&minCents=1750&maxCents=1750", statsResult{0, 1750, -1750}},
		{"combined exclusion", "?category=交通&maxCents=1250", statsResult{}},
		{"valid leap day no match", "?from=2024-02-29&to=2024-02-29", statsResult{}},
		{"int64 maximum filter", "?maxCents=9223372036854775807", statsResult{300000, 3000, 297000}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records := statsTestRecords()
			original := append([]ledger.Record(nil), records...)
			calls := 0
			handler := Stats(func() ([]ledger.Record, error) { calls++; return records, nil })
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/stats"+test.query, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
				t.Fatal("missing JSON content type")
			}
			var got statsResult
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got=%+v want=%+v", got, test.want)
			}
			var keys map[string]json.RawMessage
			_ = json.Unmarshal(response.Body.Bytes(), &keys)
			if len(keys) != 3 {
				t.Fatalf("response has extra/missing keys: %v", keys)
			}
			if calls != 1 || !reflect.DeepEqual(records, original) {
				t.Fatal("callback count or record mutation")
			}
		})
	}
}

func TestStatsRejectsBadQueriesBeforeReading(t *testing.T) {
	queries := []string{
		"?from=2026-02-29", "?to=2026-11-31", "?from=2026-1-7",
		"?from=2026-11-07T00:00:00", "?from=%202026-11-07",
		"?from=2026-11-08&to=2026-11-07", "?category=未知",
		"?minCents=-1", "?maxCents=1.5", "?minCents=1e3", "?minCents=%2B1",
		"?minCents=abc", "?maxCents=9223372036854775808",
		"?minCents=2&maxCents=1", "?category=交通&category=饮食",
		"?from=&from=2026-11-07", "?to=2026-11-07&to=2026-11-07",
		"?minCents=0&minCents=1", "?maxCents=0&maxCents=0", "?from=%zz",
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			calls := 0
			handler := Stats(func() ([]ledger.Record, error) { calls++; return nil, nil })
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/stats", nil)
			request.URL.RawQuery = strings.TrimPrefix(query, "?")
			handler.ServeHTTP(response, request)
			if response.Code != 400 || calls != 0 {
				t.Fatalf("status=%d calls=%d", response.Code, calls)
			}
			var got map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got["error"] == "" {
				t.Fatalf("not a JSON error: %s", response.Body)
			}
		})
	}
}

func TestStatsEmptyAndStorageError(t *testing.T) {
	for _, records := range [][]ledger.Record{nil, {}} {
		response := httptest.NewRecorder()
		Stats(func() ([]ledger.Record, error) { return records, nil }).ServeHTTP(response, httptest.NewRequest("GET", "/api/stats", nil))
		if response.Code != 200 || strings.TrimSpace(response.Body.String()) != `{"incomeCents":0,"expenseCents":0,"balanceCents":0}` {
			t.Fatalf("empty result: %d %s", response.Code, response.Body)
		}
	}
	response := httptest.NewRecorder()
	Stats(func() ([]ledger.Record, error) { return nil, errors.New("C:/private/path: permission denied") }).ServeHTTP(response, httptest.NewRequest("GET", "/api/stats", nil))
	if response.Code != 500 || strings.Contains(response.Body.String(), "private") {
		t.Fatal(response.Body.String())
	}
	response = httptest.NewRecorder()
	Stats(nil).ServeHTTP(response, httptest.NewRequest("GET", "/api/stats", nil))
	if response.Code != 500 {
		t.Fatal("nil dependency should not panic")
	}
}

func TestStatsOverflowAndInvalidRecords(t *testing.T) {
	for _, kind := range []string{"income", "expense"} {
		t.Run(kind, func(t *testing.T) {
			for _, extra := range []int64{0, 1} {
				records := []ledger.Record{{Type: kind, AmountCents: math.MaxInt64}}
				if extra != 0 {
					records = append(records, ledger.Record{Type: kind, AmountCents: extra})
				}
				response := httptest.NewRecorder()
				Stats(func() ([]ledger.Record, error) { return records, nil }).ServeHTTP(response, httptest.NewRequest("GET", "/api/stats", nil))
				wantStatus := 200
				if extra != 0 {
					wantStatus = 500
				}
				if response.Code != wantStatus {
					t.Fatalf("status=%d body=%s", response.Code, response.Body)
				}
			}
		})
	}
	for _, record := range []ledger.Record{{Type: "income", AmountCents: -1}, {Type: "income"}, {Type: "other", AmountCents: 1}} {
		response := httptest.NewRecorder()
		Stats(func() ([]ledger.Record, error) { return []ledger.Record{record}, nil }).ServeHTTP(response, httptest.NewRequest("GET", "/api/stats", nil))
		if response.Code != 500 {
			t.Fatalf("invalid record accepted: %+v", record)
		}
	}
}

func TestStatsMethod(t *testing.T) {
	response := httptest.NewRecorder()
	Stats(func() ([]ledger.Record, error) { t.Fatal("must not read"); return nil, nil }).ServeHTTP(response, httptest.NewRequest("POST", "/api/stats", nil))
	if response.Code != 405 || response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatal("wrong method handling")
	}
}
