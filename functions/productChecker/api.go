package main

import (
	"github.com/akijowski/target-tracker/internal/schema"
)

type TargetAPIResult struct {
	Data struct {
		FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
		Product          APIProduct          `json:"product"`
	} `json:"data"`
}

type APIFulfillmentFiats struct {
	ProductID string        `json:"product_id"`
	Locations []APILocation `json:"locations"`
}

type APILocation struct {
	LocationID                         string   `json:"location_id"`
	LocationAvailableToPromiseQuantity float64  `json:"location_available_to_promise_quantity"`
	Store                              APIStore `json:"store"`
}

type APIStore struct {
	StoreID              string                     `json:"store_id"`
	LocationName         string                     `json:"location_name"`
	MailingAddress       schema.StoreMailingAddress `json:"mailing_address"`
	MainVoicePhoneNumber string                     `json:"main_voice_phone_number"`
}

type APIProduct struct {
	TCIN        string                `json:"tcin"`
	Fulfillment APIProductFulfillment `json:"fulfillment"`
}

type APIProductFulfillment struct {
	ProductID                       string                               `json:"product_id"`
	IsOutOfStockInAllStoreLocations bool                                 `json:"is_out_of_stock_in_all_store_locations"`
	ShippingOptions                 APIProductFulfillmentShippingOptions `json:"shipping_options"`
}

type APIProductFulfillmentShippingOptions struct {
	AvailabilityStatus         string  `json:"availability_status"`
	LoyaltyAvailabilityStatus  string  `json:"loyalty_availability_status"`
	AvailableToPromiseQuantity float64 `json:"available_to_promise_quantity"`
}
