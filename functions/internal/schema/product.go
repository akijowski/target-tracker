package schema

// ProductsInput is a collection of ProductQuery data that is sent from the Step Function.
type ProductsInput struct {
	Products []Product `json:"products"`
}

// ProductQuery is a single event sent from the Step Function and contains information to query the API.
type ProductQuery struct {
	Name            string `json:"name"`
	DesiredQuantity int    `json:"desired_quantity"`
	ProductURL      string `json:"product_url"`
	TCIN            string `json:"tcin"`
}

// Product is an internal state gathering the query and result information for a given product.
type Product struct {
	ProductQuery
	Result ProductResult `json:"result"`
}

// PickupResult is a collection of stores that are available for in-store pickup.
type PickupResult struct {
	Stores      []StoreResult `json:"stores"`
	TotalStores int           `json:"total_stores"`
}

// DeliveryResult is the delivery status for the given product.
type DeliveryResult struct {
	AvailableToPromise int  `json:"available_to_promise"`
	IsAvailable        bool `json:"is_available"`
}

// ProductResult is the result of querying the API for the given product.
// `db_ttl` is included so if we persist this result, we have a TTL available.  We do it here so we don't have a lambda
// in the StepFunction that is just generating timestamps.
type ProductResult struct {
	Pickup   PickupResult   `json:"pickup,omitempty"`
	Delivery DeliveryResult `json:"delivery,omitempty"`
	DBTTL    int64          `json:"db_ttl"`
}

// StoreResult is an individual store information for the given product.
type StoreResult struct {
	AvailableToPromise int                 `json:"available"`
	LocationName       string              `json:"location_name"`
	MailingAddress     StoreMailingAddress `json:"mailing_address"`
	StoreID            string              `json:"store_id"`
}

// StoreMailingAddress is an individual store's physical address.
type StoreMailingAddress struct {
	AddressLine1 string `json:"address_line1"`
	City         string `json:"city"`
	PostalCode   string `json:"postal_code"`
	State        string `json:"state"`
}
