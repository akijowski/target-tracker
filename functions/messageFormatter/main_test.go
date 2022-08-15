package main

import (
	"context"
	"log"
	"testing"

	"github.com/akijowski/target-tracker/internal/schema"
)

func TestHandler(t *testing.T) {
	cases := map[string]struct {
		input    schema.ProductsInput
		expected MessageResult
	}{
		"No Results Returns Empty": {
			input: schema.ProductsInput{
				Products: []schema.Product{
					{
						ProductQuery: schema.ProductQuery{
							Name:            "formula",
							DesiredQuantity: 1,
							ProductURL:      "",
						},
						Result: schema.ProductResult{
							Pickup: schema.PickupResult{
								Stores:      []schema.StoreResult{},
								TotalStores: 0,
							},
						},
					},
				},
			},
			expected: MessageResult{},
		},
		"Results Returns Message": {
			input: schema.ProductsInput{
				Products: []schema.Product{
					{
						ProductQuery: schema.ProductQuery{
							Name:            "special formula",
							DesiredQuantity: 1,
							ProductURL:      "url-to-formula",
						},
						Result: schema.ProductResult{
							Pickup: schema.PickupResult{
								Stores: []schema.StoreResult{
									{
										AvailableToPromise: 3,
										LocationName:       "Denver",
										StoreID:            "1234",
										MailingAddress: schema.StoreMailingAddress{
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
			},
			expected: MessageResult{Pickup: `Product Alert!
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

`},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			logger = log.Default()
			ctx := context.Background()
			prepareTemplates()
			actual, err := handler(ctx, tt.input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if actual.Pickup != tt.expected.Pickup {
				t.Errorf("%s\n---\n%s", actual, tt.expected)
			}

			if len(actual.Shipping) > 0 {
				t.Errorf("expected shipping messsage to be empty.  Got:\n\t%s\n", actual.Shipping)
			}
		})
	}
}

func TestHandler_Shipping(t *testing.T) {
	cases := map[string]struct {
		input    schema.ProductsInput
		expected MessageResult
	}{
		"No Results Returns Empty": {
			input: schema.ProductsInput{
				Products: []schema.Product{
					{
						ProductQuery: schema.ProductQuery{
							Name:            "formula",
							DesiredQuantity: 1,
							ProductURL:      "",
						},
						Result: schema.ProductResult{
							Pickup: schema.PickupResult{
								Stores:      []schema.StoreResult{},
								TotalStores: 0,
							},
							Shipping: schema.ShippingResult{},
						},
					},
				},
			},
			expected: MessageResult{},
		},
		"Results Returns Message": {
			input: schema.ProductsInput{
				Products: []schema.Product{
					{
						ProductQuery: schema.ProductQuery{
							Name:            "special formula",
							DesiredQuantity: 1,
							ProductURL:      "url-to-formula",
						},
						Result: schema.ProductResult{
							Pickup: schema.PickupResult{
								Stores: []schema.StoreResult{},
							},
							Shipping: schema.ShippingResult{
								AvailableToPromise: 1,
								IsAvailable:        true,
							},
						},
					},
				},
			},
			expected: MessageResult{Shipping: `Product Alert!
special formula is available for online order.  1 available:
url-to-formula
`},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			logger = log.Default()
			ctx := context.Background()
			prepareTemplates()
			actual, err := handler(ctx, tt.input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if actual.Shipping != tt.expected.Shipping {
				t.Errorf("%s\n---\n%s", actual, tt.expected)
			}
			if len(actual.Pickup) > 0 {
				t.Errorf("expected pickup messsage to be empty.  Got:\n\t%s\n", actual.Pickup)
			}
		})
	}
}
