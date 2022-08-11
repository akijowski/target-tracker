package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

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
	data := HistoricalData{Count: product.Result.Pickup.TotalStores, Time: processingTime.Unix()}
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
	// checksum := sha256BytesToString(b)
	//TODO: add content type, MD5 checksum
	body := bytes.NewReader(b)
	_, err = api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(bucketName),
		Key:             aws.String(objectKey),
		Body:            body,
		ContentEncoding: aws.String("application/json"),
	})
	if err == nil {
		logger.Printf("Successfully wrote stat to S3: %s\n", b)
	}
	return err
}
