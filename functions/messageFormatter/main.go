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
	logger           *log.Logger
	pickupTemplate   *template.Template
	shippingTemplate *template.Template
)

const pickupTemplateName = "pickup_email"
const shippingTemplateName = "shipping_email"

const pickupEmailTpl string = `
{{- if gt (len .) 0 -}}
Product Alert!
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
{{- end -}}
`

const shippingEmailTpl string = `
{{- if gt (len .) 0 -}}
Product Alert!
{{ range . -}}
{{ .Name }} is available for online order.  {{ .Result.Shipping.AvailableToPromise }} available:
{{ .ProductURL }}
{{ end }}
{{ end -}}
`

// MessageResult contains formatted messages for both pickup and shipping information.
type MessageResult struct {
	Pickup   string `json:"pickup"`
	Shipping string `json:"shipping"`
}

func handler(ctx context.Context, input schema.ProductsInput) (MessageResult, error) {
	logger.Printf("input: %+v\n", input)
	availableInStore := []schema.Product{}
	availableShipping := []schema.Product{}
	result := MessageResult{}
	for _, p := range input.Products {
		if p.Result.Pickup.TotalStores > 0 {
			availableInStore = append(availableInStore, p)
		}
		if p.Result.Shipping.IsAvailable {
			availableShipping = append(availableShipping, p)
		}
	}
	logger.Printf("creating in-store pickup message for %d products\n", len(availableInStore))
	logger.Printf("creating shipping pickup message for %d products\n", len(availableShipping))
	if len(availableInStore) == 0 && len(availableShipping) == 0 {
		return result, nil
	}
	pickupMessage, err := executeTemplate(pickupTemplate, availableInStore)
	if err != nil {
		return result, err
	}
	shippingMessage, err := executeTemplate(shippingTemplate, availableShipping)
	if err != nil {
		return result, err
	}
	result.Pickup = pickupMessage
	result.Shipping = shippingMessage
	return result, nil
}

func main() {
	logger = log.Default()
	logger.SetPrefix("message_formatter ")
	logger.SetFlags(log.Lshortfile | log.Lmsgprefix)
	prepareTemplates()
	lambda.Start(handler)
}

func prepareTemplates() {
	pickupTemplate = template.Must(template.New(pickupTemplateName).Parse(pickupEmailTpl))
	shippingTemplate = template.Must(template.New(shippingTemplateName).Parse(shippingEmailTpl))
}

func executeTemplate(t *template.Template, input any) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := t.Execute(buf, input); err != nil {
		return "", err
	}
	return buf.String(), nil
}
