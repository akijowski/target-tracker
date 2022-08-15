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

	"github.com/akijowski/target-tracker/internal/schema"
)

func TestHandler(t *testing.T) {
	defaultEnv := os.Environ()

	cases := map[string]struct {
		input       schema.ProductQuery
		expected    schema.ProductResult
		apiResponse TargetAPIResult
	}{
		"No in store availability returns correctly": {
			input: schema.ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: schema.ProductResult{
				Pickup: schema.PickupResult{
					Stores:      []schema.StoreResult{},
					TotalStores: 0,
				},
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
					Product          APIProduct          `json:"product"`
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
									MailingAddress: schema.StoreMailingAddress{},
								},
							},
						},
					},
				},
			},
		},
		"In store availability returns stores": {
			input: schema.ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: schema.ProductResult{
				Pickup: schema.PickupResult{
					Stores: []schema.StoreResult{
						{
							AvailableToPromise: 10,
							LocationName:       "MockStore",
							MailingAddress:     schema.StoreMailingAddress{},
							StoreID:            "9999",
						},
					},
					TotalStores: 1,
				},
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
					Product          APIProduct          `json:"product"`
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
									MailingAddress: schema.StoreMailingAddress{},
								},
							},
							{
								LocationID:                         "0000",
								LocationAvailableToPromiseQuantity: 0,
								Store: APIStore{
									StoreID:        "0000",
									LocationName:   "EmptyStore",
									MailingAddress: schema.StoreMailingAddress{},
								},
							},
						},
					},
				},
			},
		},
		"No delivery returns correctly": {
			input: schema.ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: schema.ProductResult{
				Pickup: schema.PickupResult{
					Stores:      []schema.StoreResult{},
					TotalStores: 0,
				},
				Delivery: schema.DeliveryResult{AvailableToPromise: 0, IsAvailable: false},
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
					Product          APIProduct          `json:"product"`
				}{
					FulfillmentFiats: APIFulfillmentFiats{},
					Product: APIProduct{
						TCIN: "123456",
						Fulfillment: APIProductFulfillment{
							IsOutOfStockInAllStoreLocations: true,
							ShippingOptions: APIProductFulfillmentShippingOptions{
								AvailableToPromiseQuantity: 0,
								AvailabilityStatus:         "OUT_OF_STOCK",
							},
						},
					},
				},
			},
		},
		"Delivery available returns correctly": {
			input: schema.ProductQuery{
				Name:            "mock-product",
				TCIN:            "123456",
				DesiredQuantity: 1,
			},
			expected: schema.ProductResult{
				Pickup: schema.PickupResult{
					Stores:      []schema.StoreResult{},
					TotalStores: 0,
				},
				Delivery: schema.DeliveryResult{AvailableToPromise: 10, IsAvailable: true},
			},
			apiResponse: TargetAPIResult{
				Data: struct {
					FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
					Product          APIProduct          `json:"product"`
				}{
					FulfillmentFiats: APIFulfillmentFiats{},
					Product: APIProduct{
						TCIN: "123456",
						Fulfillment: APIProductFulfillment{
							IsOutOfStockInAllStoreLocations: true,
							ShippingOptions: APIProductFulfillmentShippingOptions{
								AvailableToPromiseQuantity: 10,
								AvailabilityStatus:         "IN_STOCK",
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
			if actual.Pickup.TotalStores != tt.expected.Pickup.TotalStores {
				t.Errorf("expected %d total stores, got: %d", tt.expected.Pickup.TotalStores, actual.Pickup.TotalStores)
			}
			if len(actual.Pickup.Stores) != len(tt.expected.Pickup.Stores) {
				t.Errorf("expected %d stores, got: %d", len(tt.expected.Pickup.Stores), len(actual.Pickup.Stores))
			}
			if actual.Delivery.AvailableToPromise != tt.expected.Delivery.AvailableToPromise {
				t.Errorf("expected %d available to promise, got: %d", tt.expected.Delivery.AvailableToPromise, actual.Delivery.AvailableToPromise)
			}
			if actual.Delivery.IsAvailable != tt.expected.Delivery.IsAvailable {
				t.Errorf("expected %v total stores, got: %v", tt.expected.Delivery.IsAvailable, actual.Delivery.IsAvailable)
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
