package main

import (
	"bytes"
	"context"
	"html/template"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	logger *log.Logger
)

const emailTpl string = `Product Alert!
{{ range . -}}
{{ .Name }}:
{{ .ProductURL }}

The following {{ .Result.TotalStores }} stores claim to have at least ({{ .DesiredQuantity }}) available:
{{ range .Result.Stores -}}
{{ .LocationName }}
	Available: {{ .AvailableToPromise }}
	StoreID: {{ .StoreID }}
	{{- with .MailingAddress }}
	Address:
		{{ .AddressLine1 }}
		{{ .City }}
		{{ .PostalCode }}
		{{ .State }}
	{{- end }}

{{ end }}
{{- end -}}
`

type ProductsInput struct {
	Products []Product `json:"products"`
}

// ProductQuery is an event sent from the Step Function and contains information to query the API.
type ProductQuery struct {
	Name            string `json:"name"`
	DesiredQuantity int    `json:"desired_quantity"`
	ProductURL      string `json:"product_url"`
}

type Product struct {
	ProductQuery
	Result ProductResult `json:"result"`
}

// ProductResult is the result of querying the API for the given product.
type ProductResult struct {
	Stores      []StoreResult `json:"stores"`
	TotalStores int           `json:"total_stores"`
}

// StoreResult is an individual store information for the given product.
type StoreResult struct {
	AvailableToPromise int                 `json:"available"`
	LocationName       string              `json:"location_name"`
	MailingAddress     StoreMailingAddress `json:"mailing_address"`
	StoreID            string              `json:"store_id"`
}

type StoreMailingAddress struct {
	AddressLine1 string `json:"address_line1"`
	City         string `json:"city"`
	PostalCode   string `json:"postal_code"`
	State        string `json:"state"`
}

func handler(ctx context.Context, input ProductsInput) (string, error) {
	logger.Printf("input: %+v\n", input)
	filtered := []Product{}
	for _, p := range input.Products {
		if p.Result.TotalStores > 0 {
			filtered = append(filtered, p)
		}
	}
	logger.Printf("creating message for %d products\n", len(filtered))
	if len(filtered) == 0 {
		return "", nil
	}
	t, err := template.New("email").Parse(emailTpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, filtered); err != nil {
		return "", err
	}
	return b.String(), nil
}

func main() {
	logger = log.Default()
	logger.SetPrefix("message_formatter ")
	logger.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	lambda.Start(handler)
}
