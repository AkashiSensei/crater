package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imroc/req/v3"
)

func billingTestClient(handler http.HandlerFunc) *Client {
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			return recorder.Result(), nil
		}
	})
	return client
}

func TestListJobBillingRoutes(t *testing.T) {
	tests := []struct {
		name     string
		options  JobBillingListOptions
		wantPath string
		wantDays string
		hasDays  bool
	}{
		{
			name:     "self",
			options:  JobBillingListOptions{Days: 30},
			wantPath: "/api/v1/vcjobs/billing",
		},
		{
			name:     "all visible",
			options:  JobBillingListOptions{All: true, Days: 14},
			wantPath: "/api/v1/vcjobs/billing/all",
			wantDays: "14",
			hasDays:  true,
		},
		{
			name:     "user takes precedence over all",
			options:  JobBillingListOptions{All: true, Username: "alice", Days: -1},
			wantPath: "/api/v1/vcjobs/billing/user/alice",
			wantDays: "-1",
			hasDays:  true,
		},
		{
			name:     "admin all",
			options:  JobBillingListOptions{Admin: true, Days: 30},
			wantPath: "/api/v1/admin/vcjobs/billing",
			wantDays: "30",
			hasDays:  true,
		},
		{
			name:     "admin user",
			options:  JobBillingListOptions{Admin: true, Username: "bob", Days: 7},
			wantPath: "/api/v1/admin/vcjobs/billing/user/bob",
			wantDays: "7",
			hasDays:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := billingTestClient(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", request.Method)
				}
				if request.URL.Path != test.wantPath {
					t.Errorf("path = %s, want %s", request.URL.Path, test.wantPath)
				}
				days, present := request.URL.Query()["days"]
				if present != test.hasDays {
					t.Errorf("days present = %t, want %t", present, test.hasDays)
				}
				if test.hasDays && (len(days) != 1 || days[0] != test.wantDays) {
					t.Errorf("days = %v, want %q", days, test.wantDays)
				}
				writer.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(writer).Encode(Response[[]JobBilling]{
					Data: []JobBilling{{
						Name: "training", JobName: "train-alice-1", BilledPointsTotal: 12.5,
					}},
				})
			})

			records, err := client.ListJobBilling(test.options)
			if err != nil {
				t.Fatalf("ListJobBilling: %v", err)
			}
			if len(records) != 1 ||
				records[0].JobName != "train-alice-1" ||
				records[0].BilledPointsTotal != 12.5 {
				t.Fatalf("records = %#v", records)
			}
		})
	}
}

func TestListJobBillingNormalizesNullData(t *testing.T) {
	client := billingTestClient(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[[]JobBilling]{Data: nil})
	})

	records, err := client.ListJobBilling(JobBillingListOptions{})
	if err != nil {
		t.Fatalf("ListJobBilling: %v", err)
	}
	if records == nil || len(records) != 0 {
		t.Fatalf("records = %#v, want non-nil empty slice", records)
	}
}
