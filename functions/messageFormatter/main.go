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

func handler(ctx context.Context, input schema.ProductsInput) (string, error) {
	logger.Printf("input: %+v\n", input)
	filtered := []schema.Product{}
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
	logger.SetFlags(log.Lshortfile | log.Lmsgprefix)
	lambda.Start(handler)
}
