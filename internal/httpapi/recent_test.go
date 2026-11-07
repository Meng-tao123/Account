package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"personal-ledger/internal/ledger"
)

func TestRecentRecords(t *testing.T) {
	tests := []struct {
		name      string
		records   []ledger.Record
		query     string
		wantCount int
		wantFirst int64
		wantLast  int64
	}{
		{name: "nil ledger"},
		{name: "empty ledger", records: []ledger.Record{}},
		{name: "one record", records: recentTestRecords(1), wantCount: 1, wantFirst: 1, wantLast: 1},
		{name: "fewer than ten", records: recentTestRecords(3), wantCount: 3, wantFirst: 3, wantLast: 1},
		{name: "exactly ten", records: recentTestRecords(10), wantCount: 10, wantFirst: 10, wantLast: 1},
		{name: "default keeps latest ten", records: recentTestRecords(12), wantCount: 10, wantFirst: 12, wantLast: 3},
		{name: "limit one", records: recentTestRecords(12), query: "?limit=1", wantCount: 1, wantFirst: 12, wantLast: 12},
		{name: "explicit ten", records: recentTestRecords(12), query: "?limit=10", wantCount: 10, wantFirst: 12, wantLast: 3},
		{name: "limit one hundred", records: recentTestRecords(105), query: "?limit=100", wantCount: 100, wantFirst: 105, wantLast: 6},
		{name: "limit exceeds ledger size", records: recentTestRecords(3), query: "?limit=100", wantCount: 3, wantFirst: 3, wantLast: 1},
		{name: "empty ledger with limit", query: "?limit=100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.records)
			calls := 0
			handler := Recent(func() ([]ledger.Record, error) {
				calls++
				return tt.records, nil
			})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent"+tt.query, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
			}
			recentAssertJSON(t, response)
			var got []ledger.Record
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode record array: %v", err)
			}
			if got == nil {
				t.Fatal("response must be an array, including [] for an empty ledger")
			}
			if len(got) != tt.wantCount {
				t.Fatalf("record count = %d, want %d", len(got), tt.wantCount)
			}
			if len(got) > 0 {
				if got[0].AmountCents != tt.wantFirst || got[len(got)-1].AmountCents != tt.wantLast {
					t.Errorf("first/last amounts = %d/%d, want %d/%d", got[0].AmountCents, got[len(got)-1].AmountCents, tt.wantFirst, tt.wantLast)
				}
				for i, record := range got {
					want := tt.records[len(tt.records)-1-i]
					if record != want {
						t.Errorf("record %d = %+v, want %+v", i, record, want)
					}
				}
			} else if strings.TrimSpace(response.Body.String()) != "[]" {
				t.Errorf("empty response = %q, want []", response.Body.String())
			}
			if calls != 1 {
				t.Errorf("listRecords calls = %d, want 1", calls)
			}
			if !slices.Equal(tt.records, before) {
				t.Errorf("source slice changed: got %+v, before %+v", tt.records, before)
			}
		})
	}
}

func TestRecentInvalidLimit(t *testing.T) {
	queries := []string{
		"?limit=0", "?limit=101", "?limit=-1", "?limit=1.5",
		"?limit=abc", "?limit=", "?limit", "?limit=%20",
		"?limit=%2010%20", "?limit=1e1", "?limit=0x10",
		"?limit=999999999999999999999999999999999999",
		"?limit=1&limit=2", "?limit=&limit=10", "?limit=10&limit=",
		"?limit=%ZZ", "?limit=1;limit=2",
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			handler := Recent(func() ([]ledger.Record, error) {
				t.Fatal("invalid limit must be rejected before reading storage")
				return nil, nil
			})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent"+query, nil))
			recentAssertError(t, response, http.StatusBadRequest)
		})
	}
}

func TestRecentBackdatedRecordAndRepeatedReads(t *testing.T) {
	records := recentTestRecords(3)
	records[0].Date = "2026-11-06"
	records[1].Date = "2026-11-07"
	records[2].Date = "2020-01-01"
	records[2].Note = "新录入的旧日期补账"
	before := slices.Clone(records)
	handler := Recent(func() ([]ledger.Record, error) { return records, nil })
	want := []ledger.Record{before[2], before[1], before[0]}
	for attempt := 0; attempt < 2; attempt++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.Code)
		}
		var got []ledger.Record
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("attempt %d: got %+v, want insertion order %+v", attempt, got, want)
		}
		if !slices.Equal(records, before) {
			t.Fatalf("source slice changed: %+v", records)
		}
	}
}

func TestRecentReadsCurrentRecords(t *testing.T) {
	records := recentTestRecords(1)
	calls := 0
	handler := Recent(func() ([]ledger.Record, error) {
		calls++
		return records, nil
	})
	for count := 1; count <= 2; count++ {
		records = recentTestRecords(count)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent?limit=1", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.Code)
		}
		var got []ledger.Record
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != records[count-1] {
			t.Fatalf("response did not reflect latest storage data: %+v", got)
		}
	}
	if calls != 2 {
		t.Errorf("listRecords calls = %d, want 2", calls)
	}
}

func TestRecentStorageError(t *testing.T) {
	handler := Recent(func() ([]ledger.Record, error) {
		// A partial result must not be returned as a successful response.
		return recentTestRecords(1), errors.New("storage unavailable")
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent", nil))
	recentAssertError(t, response, http.StatusInternalServerError)
}

func TestRecentJSONFields(t *testing.T) {
	record := ledger.Record{Date: "2026-11-07", Type: "expense", AmountCents: 1250, Category: "饮食", Note: "午饭"}
	handler := Recent(func() ([]ledger.Record, error) { return []ledger.Record{record}, nil })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/records/recent", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{{"date": "2026-11-07", "type": "expense", "amountCents": float64(1250), "category": "饮食", "note": "午饭"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON fields = %#v, want %#v", got, want)
	}
}

func recentTestRecords(count int) []ledger.Record {
	records := make([]ledger.Record, count)
	for i := range records {
		records[i] = ledger.Record{
			Date: "2026-11-07", Type: "expense", AmountCents: int64(i + 1),
			Category: "饮食", Note: fmt.Sprintf("记录 %d", i+1),
		}
	}
	return records
}

func recentAssertJSON(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
}

func recentAssertError(t *testing.T, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	recentAssertJSON(t, response)
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON error: %v", err)
	}
	var message string
	if err := json.Unmarshal(body["error"], &message); err != nil || message == "" || len(body) != 1 {
		t.Errorf("want a single nonempty string error, got %s", response.Body.String())
	}
}
