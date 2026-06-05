package utils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	ssoConfig "sso.pelajarnumagetan.or.id/internal/config"
)

var r2Client *s3.Client

func getR2Client() (*s3.Client, error) {
	if r2Client != nil {
		return r2Client, nil
	}

	cfgData := ssoConfig.Get()
	if cfgData.R2AccessKeyID == "" || cfgData.R2SecretAccessKey == "" || cfgData.R2Endpoint == "" {
		return nil, errors.New("Cloudflare R2 is not fully configured in env")
	}

	// Initialize AWS SDK config for R2
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfgData.R2AccessKeyID,
			cfgData.R2SecretAccessKey,
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	// Create S3 client specifying custom base endpoint
	r2Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfgData.R2Endpoint)
	})

	return r2Client, nil
}

// UploadToR2 uploads data to Cloudflare R2 bucket.
func UploadToR2(ctx context.Context, key string, data []byte, contentType string) error {
	client, err := getR2Client()
	if err != nil {
		return err
	}

	cfgData := ssoConfig.Get()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfgData.R2Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("Failed to upload %s to R2: %v", key, err)
		return err
	}

	return nil
}

// GetFromR2 retrieves data from Cloudflare R2 bucket.
func GetFromR2(ctx context.Context, key string) ([]byte, error) {
	client, err := getR2Client()
	if err != nil {
		return nil, err
	}

	cfgData := ssoConfig.Get()
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfgData.R2Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to get %s from R2: %v", key, err)
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DeleteFromR2 deletes data from Cloudflare R2 bucket.
func DeleteFromR2(ctx context.Context, key string) error {
	client, err := getR2Client()
	if err != nil {
		return err
	}

	cfgData := ssoConfig.Get()
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(cfgData.R2Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to delete %s from R2: %v", key, err)
		return err
	}

	return nil
}
