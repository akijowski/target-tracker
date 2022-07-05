package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
)

const (
	s3URIEnv       = "S3_URI_OVERRIDE"
	s3NoSuchKeyErr = "NoSuchKey"
	bucketNameEnv  = "STATS_BUCKET_NAME"
	objectKey      = "historical_stats.json"
)

var (
	s3APIClient    S3API
	logger         *log.Logger
	processingTime time.Time
)

func handler(ctx context.Context, input schema.ProductsInput) error {
	processingTime = time.Now()
	bucketName := os.Getenv(bucketNameEnv)
	logger.Printf("found bucket name: %s\n", bucketName)
	logger.Printf("input: %+v\n", input)
	// getCurrentStatsOrEmptyStruct
	stats, err := getCurrentStatsOrEmpty(ctx, s3APIClient, bucketName)
	if err != nil {
		return err
	}
	// ifOverOneWeekMakeNewStruct
	// TODO: time cutoff
	checkStatsAge(stats, processingTime)
	// addHistoricalStats
	for _, product := range input.Products {
		addHistoricalData(stats, product)
	}
	// update products
	stats.Products = input.Products
	// saveToS3
	return saveStatsToS3(ctx, s3APIClient, bucketName, stats)
}

func main() {
	logger = log.Default()
	logger.SetPrefix("historical_stats ")
	logger.SetFlags(log.Lshortfile | log.Lmsgprefix)
	client, err := configureS3Client()
	s3APIClient = client
	if err != nil {
		panic(err)
	}
	lambda.Start(handler)
}

func configureS3Client() (S3API, error) {
	ctx := context.Background()
	if os.Getenv(s3URIEnv) != "" {
		logger.Printf("Custom S3 URI found: %s\n", os.Getenv(s3URIEnv))
		endpointCfg := config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: os.Getenv(s3URIEnv),
			}, nil
		}))
		cfg, err := config.LoadDefaultConfig(ctx, endpointCfg)
		if err != nil {
			return nil, err
		}
		// Localstack s3 requires path style
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		}), nil
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")
		awsv2.AWSV2Instrumentor(&cfg.APIOptions)
		return s3.NewFromConfig(cfg), nil
	}
}

// func sha256BytesToString(b []byte) string {
// 	sha256Hash := sha256.New()
// 	return base64.StdEncoding.EncodeToString(sha256Hash.Sum(b))
// }
