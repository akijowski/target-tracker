package main

import (
	"bytes"
	"context"
	"html/template"
	"log"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	logger *log.Logger
)

const pickupEmailTpl string = `Product Alert!
{{ range . -}}
{{ .Name }}:
{{ .ProductURL }}

The following {{ .Result.Pickup.TotalStores }} stores claim to have at least ({{ .DesiredQuantity }}) available:
{{ range .Result.Pickup.Stores -}}
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

// MessageResult contains formatted messages for both pickup and delivery information.
type MessageResult struct {
	Pickup   string `json:"pickup"`
	Delivery string `json:"delivery"`
}

func handler(ctx context.Context, input schema.ProductsInput) (MessageResult, error) {
	logger.Printf("input: %+v\n", input)
	availableInStore := []schema.Product{}
	for _, p := range input.Products {
		if p.Result.Pickup.TotalStores > 0 {
			availableInStore = append(availableInStore, p)
		}
	}
	logger.Printf("creating in-store pickup message for %d products\n", len(availableInStore))
	if len(availableInStore) == 0 {
		return MessageResult{}, nil
	}
	t, err := template.New("email").Parse(pickupEmailTpl)
	if err != nil {
		return MessageResult{}, err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, availableInStore); err != nil {
		return MessageResult{}, err
	}
	return MessageResult{Pickup: b.String()}, nil
}

func main() {
	logger = log.Default()
	logger.SetPrefix("message_formatter ")
	logger.SetFlags(log.Lshortfile | log.Lmsgprefix)
	lambda.Start(handler)
}
