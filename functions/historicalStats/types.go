package main

import (
	"context"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
