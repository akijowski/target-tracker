package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"time"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/smithy-go"
)

type S3API interface {
	S3GetObjectAPI
	S3PutObjectAPI
}

type S3PutObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type HistoricalStats struct {
	CreatedAt     int64            `json:"created_at"`
	LastUpdatedAt int64            `json:"last_updated_at"`
	Products      []schema.Product `json:"products"`
	History       []HistoricalStat `json:"history"`
}

type HistoricalStat struct {
	ProductName string           `json:"product_name"`
	Data        []HistoricalData `json:"data"`
}

type HistoricalData struct {
	Time  int64 `json:"time"`
	Count int   `json:"count"`
}

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

func getCurrentStatsOrEmpty(ctx context.Context, api S3GetObjectAPI, bucketName string) (*HistoricalStats, error) {
	out, err := api.GetObject(ctx, &s3.GetObjectInput{
		Bucket:              aws.String(bucketName),
		Key:                 aws.String(objectKey),
		ChecksumMode:        types.ChecksumModeEnabled,
		ResponseContentType: aws.String("application/json"),
	})
	var ae smithy.APIError
	if errors.As(err, &ae) {
		logger.Printf("S3 error: %s", ae)
		logger.Printf("ae: %#v", ae)
		if ae.ErrorCode() == s3NoSuchKeyErr {
			return &HistoricalStats{CreatedAt: processingTime.Unix()}, nil
		} else {
			return nil, err
		}
	}
	defer out.Body.Close()
	var hs HistoricalStats
	if b, err := io.ReadAll(out.Body); err != nil {
		return nil, err
	} else {
		if err := json.Unmarshal(b, &hs); err != nil {
			return nil, err
		}
	}
	return &hs, nil
}

func checkStatsAge(stats *HistoricalStats, procTime time.Time) {
	createdAt := time.Unix(stats.CreatedAt, 0).Round(time.Hour)
	oneWeekAgo := procTime.Add(-7 * 24 * time.Hour).Round(time.Hour)
	// logger.Printf("created at: %v\nweek ago: %v\n", createdAt, oneWeekAgo)
	if oneWeekAgo.After(createdAt) {
		logger.Println("stats are old and must be recreated")
		*stats = HistoricalStats{CreatedAt: procTime.Unix()}
	}
}

func addHistoricalData(stats *HistoricalStats, product schema.Product) {
	data := HistoricalData{Count: product.Result.TotalStores, Time: processingTime.Unix()}
	statIdx := -1
	for i, existingStat := range stats.History {
		if existingStat.ProductName == product.ProductQuery.Name {
			statIdx = i
			break
		}
	}
	if statIdx == -1 {
		newStat := HistoricalStat{ProductName: product.Name, Data: []HistoricalData{data}}
		stats.History = append(stats.History, newStat)
	} else {
		stats.History[statIdx].Data = append(stats.History[statIdx].Data, data)
	}
	stats.LastUpdatedAt = processingTime.Unix()
}

func saveStatsToS3(ctx context.Context, api S3PutObjectAPI, bucketName string, stats *HistoricalStats) error {
	b, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	checksum := sha256BytesToString(b)
	body := bytes.NewReader(b)
	_, err = api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(bucketName),
		Key:               aws.String(objectKey),
		Body:              body,
		ContentEncoding:   aws.String("application/json"),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		ChecksumSHA256:    aws.String(checksum),
	})
	logger.Printf("Successfully wrote stat to S3: %s\n", b)
	return err
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

func sha256BytesToString(b []byte) string {
	sha256Hash := sha256.New()
	return base64.StdEncoding.EncodeToString(sha256Hash.Sum(b))
}
