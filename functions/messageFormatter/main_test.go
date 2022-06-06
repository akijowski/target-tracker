package main

import (
	"context"
	"log"
	"testing"
)

func TestHandler(t *testing.T) {
	cases := map[string]struct {
		input    ProductsInput
		expected string
	}{
		"No Results Returns Empty": {
			input: ProductsInput{
				Products: []Product{
					{
						ProductQuery: ProductQuery{
							Name:            "formula",
							DesiredQuantity: 1,
							ProductURL:      "",
						},
						Result: ProductResult{
							Stores: []StoreResult{
								{AvailableToPromise: 0, LocationName: "Denver", StoreID: "1234"},
							},
							TotalStores: 0,
						},
					},
				},
			},
			expected: "",
		},
		"Results Returns Message": {
			input: ProductsInput{
				Products: []Product{
					{
						ProductQuery: ProductQuery{
							Name:            "special formula",
							DesiredQuantity: 1,
							ProductURL:      "url-to-formula",
						},
						Result: ProductResult{
							Stores: []StoreResult{
								{
									AvailableToPromise: 3,
									LocationName:       "Denver",
									StoreID:            "1234",
									MailingAddress: StoreMailingAddress{
										AddressLine1: "123 Main St",
										City:         "Denver",
										State:        "Colorado",
										PostalCode:   "80100",
									},
								},
							},
							TotalStores: 1,
						},
					},
				},
			},
			expected: `Product Alert!
special formula:
url-to-formula

The following 1 stores claim to have at least (1) available:
Denver
	Available: 3
	StoreID: 1234
	Address:
		123 Main St
		Denver
		80100
		Colorado

`,
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			logger = log.Default()
			ctx := context.Background()
			actual, err := handler(ctx, tt.input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if actual != tt.expected {
				t.Errorf("%s\n---\n%s", actual, tt.expected)
			}
		})
	}
}
