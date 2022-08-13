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
	deliveryTemplate *template.Template
)

const pickupTemplateName = "pickup_email"
const deliveryTemplateName = "delivery_email"

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

const deliveryEmailTpl string = `
{{- if gt (len .) 0 -}}
Product Alert!
{{ range . -}}
{{ .Name }} is available for online order.  {{ .Result.Delivery.AvailableToPromise }} available:
{{ .ProductURL }}
{{- end }}
{{ end -}}
`

// MessageResult contains formatted messages for both pickup and delivery information.
type MessageResult struct {
	Pickup   string `json:"pickup"`
	Delivery string `json:"delivery"`
}

func handler(ctx context.Context, input schema.ProductsInput) (MessageResult, error) {
	logger.Printf("input: %+v\n", input)
	availableInStore := []schema.Product{}
	availableDelivery := []schema.Product{}
	result := MessageResult{}
	for _, p := range input.Products {
		if p.Result.Pickup.TotalStores > 0 {
			availableInStore = append(availableInStore, p)
		}
		if p.Result.Delivery.IsAvailable {
			availableDelivery = append(availableDelivery, p)
		}
	}
	logger.Printf("creating in-store pickup message for %d products\n", len(availableInStore))
	logger.Printf("creating delivery pickup message for %d products\n", len(availableDelivery))
	if len(availableInStore) == 0 && len(availableDelivery) == 0 {
		return result, nil
	}
	pickupMessage, err := executeTemplate(pickupTemplate, availableInStore)
	if err != nil {
		return result, err
	}
	deliveryMessage, err := executeTemplate(deliveryTemplate, availableDelivery)
	if err != nil {
		return result, err
	}
	result.Pickup = pickupMessage
	result.Delivery = deliveryMessage
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
	deliveryTemplate = template.Must(template.New(deliveryTemplateName).Parse(deliveryEmailTpl))
}

func executeTemplate(t *template.Template, input any) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := t.Execute(buf, input); err != nil {
		return "", err
	}
	return buf.String(), nil
}
