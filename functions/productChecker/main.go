package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
)

const (
	URIEnvKey string = "API_URI"
	basePath  string = "/redsky_aggregations/v1/web_platform/fiats_v1"
	URLKey    string = "9f36aeafbe60771e321a7cc95a78140772ab3e96"
	// 1 week
	TTLOffset int64 = 7 * 24 * 3600
)

var logger *log.Logger
var client *http.Client = http.DefaultClient

func handler(ctx context.Context, productQuery schema.ProductQuery) (schema.ProductResult, error) {
	logger.Printf("product query: %+v\n", productQuery)
	if productQuery.DesiredQuantity == 0 {
		productQuery.DesiredQuantity = 1
	}
	resp, err := doAPIRequest(ctx, productQuery)
	if err != nil {
		return schema.ProductResult{}, err
	}
	allStores := storeResultsFromLocations(resp.Data.FulfillmentFiats.Locations)
	logger.Printf("API returned results from (%d) stores\n", len(allStores))
	filtered := []schema.StoreResult{}
	for _, s := range allStores {
		if s.AvailableToPromise > 0 {
			filtered = append(filtered, s)
		}
	}
	logger.Printf("(%d) stores with product\n", len(filtered))
	return schema.ProductResult{
		Pickup:   schema.PickupResult{Stores: filtered, TotalStores: len(filtered)},
		Delivery: schema.DeliveryResult{},
		DBTTL:    time.Now().Unix() + TTLOffset,
	}, nil
}

func main() {
	logger = log.Default()
	logger.SetPrefix("product_checker ")
	logger.SetFlags(log.Lshortfile | log.Lmsgprefix)
	client = xray.Client(client)
	lambda.Start(handler)
}

func marshalURL(pq schema.ProductQuery) string {
	host := os.Getenv(URIEnvKey)
	q := url.Values{}
	q.Set("key", URLKey)
	q.Set("nearby", "80134")
	q.Set("radius", "50")
	q.Set("limit", "20")
	q.Set("include_only_available_stores", "false")
	q.Set("tcin", pq.TCIN)
	q.Set("requested_quantity", strconv.Itoa(int(pq.DesiredQuantity)))

	return fmt.Sprintf("%s%s?%s", host, basePath, q.Encode())
}

func storeResultsFromLocations(locs []APILocation) []schema.StoreResult {
	r := []schema.StoreResult{}
	for _, l := range locs {
		r = append(r, schema.StoreResult{
			AvailableToPromise: int(l.LocationAvailableToPromiseQuantity),
			LocationName:       l.Store.LocationName,
			MailingAddress:     l.Store.MailingAddress,
			StoreID:            l.LocationID,
		})
	}
	return r
}

func doAPIRequest(ctx context.Context, pq schema.ProductQuery) (result TargetAPIResult, err error) {
	url := marshalURL(pq)
	// logger.Printf("url: %s\n", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &result)
	return
}
