package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	defaultEnv := os.Environ()

	cases := map[string]struct {
		input       ProductQuery
		expected    ProductResult
		apiResponse TargetAPIResult
	}{
		"No availability returns correctly": {
			input: ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: ProductResult{
				Stores:      []StoreResult{},
				TotalStores: 0,
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
				}{
					FulfillmentFiats: APIFulfillmentFiats{
						ProductID: "123456",
						Locations: []APILocation{
							{
								LocationID:                         "9999",
								LocationAvailableToPromiseQuantity: 0,
								Store: APIStore{
									StoreID:        "9999",
									LocationName:   "MockStore",
									MailingAddress: StoreMailingAddress{},
								},
							},
						},
					},
				},
			},
		},
		"Availability returns stores": {
			input: ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: ProductResult{
				Stores: []StoreResult{
					{
						AvailableToPromise: 10,
						LocationName:       "MockStore",
						MailingAddress:     StoreMailingAddress{},
						StoreID:            "9999",
					},
				},
				TotalStores: 1,
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
				}{
					FulfillmentFiats: APIFulfillmentFiats{
						ProductID: "123456",
						Locations: []APILocation{
							{
								LocationID:                         "9999",
								LocationAvailableToPromiseQuantity: 10,
								Store: APIStore{
									StoreID:        "9999",
									LocationName:   "MockStore",
									MailingAddress: StoreMailingAddress{},
								},
							},
							{
								LocationID:                         "0000",
								LocationAvailableToPromiseQuantity: 0,
								Store: APIStore{
									StoreID:        "0000",
									LocationName:   "EmptyStore",
									MailingAddress: StoreMailingAddress{},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			logger = log.Default()
			ctx := context.Background()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Helper()
				w.WriteHeader(http.StatusOK)
				w.Header().Add("Content-Type", "application/json")
				b, err := json.Marshal(tt.apiResponse)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				_, err = w.Write(b)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}))
			defer func() {
				mockServer.Close()
				resetEnv(defaultEnv)
			}()

			if err := os.Setenv(URIEnvKey, mockServer.URL); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			actual, err := handler(ctx, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			approxTTL := time.Now().Unix() - 1
			if actual.DBTTL < approxTTL {
				t.Errorf("DBTTL is not correct: %d, should be greater than %d", actual.DBTTL, approxTTL)
			}
			if actual.TotalStores != tt.expected.TotalStores {
				t.Errorf("expected %d total stores, got: %d", tt.expected.TotalStores, actual.TotalStores)
			}
			if len(actual.Stores) != len(tt.expected.Stores) {
				t.Errorf("expected %d stores, got: %d", len(tt.expected.Stores), len(actual.Stores))
			}
		})
	}
}

func resetEnv(environ []string) {
	for _, env := range environ {
		s := strings.Split(env, "=")
		os.Setenv(s[0], s[1])
	}
}
