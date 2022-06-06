package main

type TargetAPIResult struct {
	Data struct {
		FulfillmentFiats APIFulfillmentFiats `json:"fulfillment_fiats"`
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
	StoreID              string              `json:"store_id"`
	LocationName         string              `json:"location_name"`
	MailingAddress       StoreMailingAddress `json:"mailing_address"`
	MainVoicePhoneNumber string              `json:"main_voice_phone_number"`
}

type StoreMailingAddress struct {
	AddressLine1 string `json:"address_line1"`
	City         string `json:"city"`
	PostalCode   string `json:"postal_code"`
	State        string `json:"state"`
}
