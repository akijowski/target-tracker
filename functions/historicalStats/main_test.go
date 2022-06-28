package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"reflect"
	"testing"
	"testing/iotest"
	"time"

	"github.com/akijowski/target-tracker/internal/schema"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/davecgh/go-spew/spew"
)

type mockS3PutObjectAPI func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)

func (m mockS3PutObjectAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m(ctx, params, optFns...)
}

type mockS3GetObjectAPI func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)

func (m mockS3GetObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestGetCurrentStatsOrEmpty(t *testing.T) {
	logger = log.Default()
	now := time.Now()
	processingTime = now
	nowUnix := now.Unix()
	cases := map[string]struct {
		expected    *HistoricalStats
		expectedErr error
		api         func(t *testing.T) S3GetObjectAPI
	}{
		"Successful API returns stats": {
			expected: &HistoricalStats{
				CreatedAt:     nowUnix,
				LastUpdatedAt: nowUnix,
				Products: []schema.Product{
					{
						ProductQuery: schema.ProductQuery{
							Name:            "formula",
							DesiredQuantity: 1,
						},
					},
				},
			},
			api: func(t *testing.T) S3GetObjectAPI {
				return mockS3GetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					validateGetObjectParams(t, params)
					b, err := json.Marshal(HistoricalStats{
						CreatedAt:     nowUnix,
						LastUpdatedAt: nowUnix,
						Products: []schema.Product{
							{
								ProductQuery: schema.ProductQuery{
									Name:            "formula",
									DesiredQuantity: 1,
								},
							},
						},
					})
					if err != nil {
						t.Fatalf("unexpected error: %s", err)
					}
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(b)),
					}, nil
				})
			},
		},
		"IO error returns error": {
			expectedErr: errors.New("Can't read this"),
			api: func(t *testing.T) S3GetObjectAPI {
				return mockS3GetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					validateGetObjectParams(t, params)
					expectedErr := errors.New("Can't read this")
					return &s3.GetObjectOutput{
						Body: io.NopCloser(iotest.ErrReader(expectedErr)),
					}, nil
				})
			},
		},
		"S3 404 error returns empty stats": {
			expected: &HistoricalStats{CreatedAt: nowUnix},
			api: func(t *testing.T) S3GetObjectAPI {
				return mockS3GetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					validateGetObjectParams(t, params)
					ae := &smithy.GenericAPIError{
						Code:    "404",
						Message: "Object not found",
						Fault:   smithy.FaultUnknown,
					}
					return nil, ae
				})
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			api := tt.api(t)

			actual, err := getCurrentStatsOrEmpty(ctx, api, name)

			if err != nil {
				if tt.expectedErr == nil {
					t.Fatalf("unexpected error: %s\n", err)
				} else {
					if err.Error() != tt.expectedErr.Error() {
						t.Errorf("wanted error: %s\ngot: %s\n", tt.expectedErr, err)
					}
				}
			}
			if tt.expected != nil {
				if !reflect.DeepEqual(tt.expected, actual) {
					t.Error(spew.Printf("%v\n%v", tt.expected, actual))
				}
			}
		})
	}
}

func TestSaveStatsToS3(t *testing.T) {
	bucketName := "test-bucket"
	stats := &HistoricalStats{
		CreatedAt:     time.Now().Unix(),
		LastUpdatedAt: time.Now().Unix(),
		Products: []schema.Product{
			{ProductQuery: schema.ProductQuery{Name: "formula"}},
		},
	}
	cases := map[string]struct {
		expectedErr error
		api         func(t *testing.T) S3PutObjectAPI
	}{
		"No API error returns no error": {
			api: func(t *testing.T) S3PutObjectAPI {
				return mockS3PutObjectAPI(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					t.Helper()
					validatePutObjectParams(t, params)
					return nil, nil
				})
			},
		},
		"API error returns error": {
			expectedErr: errors.New("api error 404: Object not found"),
			api: func(t *testing.T) S3PutObjectAPI {
				return mockS3PutObjectAPI(func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					t.Helper()
					validatePutObjectParams(t, params)
					ae := &smithy.GenericAPIError{
						Code:    "404",
						Message: "Object not found",
						Fault:   smithy.FaultUnknown,
					}
					return nil, ae
				})
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			api := tt.api(t)
			err := saveStatsToS3(ctx, api, bucketName, stats)

			if err != nil {
				if tt.expectedErr == nil {
					t.Fatalf("unexpected error: %s\n", err)
				} else {
					if err.Error() != tt.expectedErr.Error() {
						t.Errorf("wanted error: %s\ngot: %s\n", tt.expectedErr, err)
					}
				}
			}
		})
	}
}

func TestCheckStatsAge(t *testing.T) {
	now := time.Now()
	processingTime = now
	logger = log.Default()
	cases := map[string]struct {
		statsIn  *HistoricalStats
		statsOut *HistoricalStats
		procTime time.Time
	}{
		"old stats are replaced": {
			procTime: now,
			statsIn:  &HistoricalStats{CreatedAt: now.Add(-8 * 24 * time.Hour).Unix()},
			statsOut: &HistoricalStats{CreatedAt: now.Unix()},
		},
		"new stats remain": {
			procTime: now,
			statsIn: &HistoricalStats{
				CreatedAt:     now.Add(-10 * time.Minute).Unix(),
				LastUpdatedAt: now.Unix(),
				History: []HistoricalStat{
					{
						ProductName: "formula",
						Data:        []HistoricalData{{Time: now.Unix(), Count: 1}},
					},
				},
			},
			statsOut: &HistoricalStats{
				CreatedAt:     now.Add(-10 * time.Minute).Unix(),
				LastUpdatedAt: now.Unix(),
				History: []HistoricalStat{
					{
						ProductName: "formula",
						Data:        []HistoricalData{{Time: now.Unix(), Count: 1}},
					},
				},
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			checkStatsAge(tt.statsIn, tt.procTime)
			if !reflect.DeepEqual(tt.statsIn, tt.statsOut) {
				t.Error(spew.Printf("%v\n%v\n", tt.statsIn, tt.statsOut))
			}
		})
	}
}

func TestAddHistoricalData(t *testing.T) {
	now := time.Now()
	processingTime = now
	cases := map[string]struct {
		stats   *HistoricalStats
		product schema.Product
		history []HistoricalStat
	}{
		"new product is added": {
			history: []HistoricalStat{
				{
					ProductName: "other formula",
				},
				{
					ProductName: "formula",
					Data: []HistoricalData{
						{
							Time:  now.Unix(),
							Count: 2,
						},
					},
				},
			},
			product: schema.Product{
				ProductQuery: schema.ProductQuery{
					Name: "formula",
				},
				Result: schema.ProductResult{
					TotalStores: 2,
				},
			},
			stats: &HistoricalStats{
				History: []HistoricalStat{
					{
						ProductName: "other formula",
					},
				},
			},
		},
		"existing product is updated": {
			history: []HistoricalStat{
				{
					ProductName: "formula",
					Data: []HistoricalData{
						{
							Time:  now.Add(-1 * time.Hour).Unix(),
							Count: 4,
						},
						{
							Time:  now.Unix(),
							Count: 2,
						},
					},
				},
				{
					ProductName: "other formula",
				},
			},
			product: schema.Product{
				ProductQuery: schema.ProductQuery{
					Name: "formula",
				},
				Result: schema.ProductResult{
					TotalStores: 2,
				},
			},
			stats: &HistoricalStats{
				History: []HistoricalStat{
					{
						ProductName: "formula",
						Data: []HistoricalData{
							{
								Time:  now.Add(-1 * time.Hour).Unix(),
								Count: 4,
							},
						},
					},
					{
						ProductName: "other formula",
					},
				},
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			addHistoricalData(tt.stats, tt.product)
			if len(tt.stats.History) != len(tt.history) {
				t.Fatalf("history was not updated.  Got %d wanted %d", len(tt.stats.History), len(tt.history))
			}
			for i, h := range tt.stats.History {
				if !reflect.DeepEqual(h, tt.history[i]) {
					t.Error(spew.Printf("%#v\n%#v\n", h, tt.history[i]))
				}
			}
		})
	}
}

func validateGetObjectParams(t testing.TB, params *s3.GetObjectInput) {
	if params.Bucket == nil || params.Key == nil {
		t.Log(spew.Printf("bucket: %v\nkey:%v\n", params.Bucket, params.Key))
		t.Error("params.Bucket and params.Key cannot be nil")
	}
}

func validatePutObjectParams(t testing.TB, params *s3.PutObjectInput) {
	if params.Bucket == nil || params.Key == nil || params.Body == nil {
		t.Log(spew.Printf("bucket: %v\nkey:%v\nbody: %v\n", params.Bucket, params.Key, params.Body))
		t.Error("params.Bucket and params.Key and params.Body cannot be nil")
	}
}
