package minio

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"

	"tibsphereai/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/exp/slog"
)

type MinIO struct {
	Client *minio.Client
	Cnf    *config.Config
}

func MinIOConnect(cnf *config.Config) (*MinIO, error) {
	endpoint := cnf.MINIO_ENDPOINT + cnf.MINIO_PORT
	accessKeyID := cnf.MINIO_ACCESS_KEY
	secretAccessKey := cnf.MINIO_SECRET_KEY
	bucketName := cnf.MINIO_BUCKET_NAME

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		slog.Error("Failed to connect to MinIO: %v", err)
		return nil, err
	}

	// Create the bucket if it doesn't exist
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check if the bucket already exists
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			slog.Warn("Bucket already exists: %s\n", bucketName)
		} else {
			slog.Error("Error while making bucket %s: %v\n", bucketName, err)
		}
	} else {
		slog.Info("Successfully created bucket: %s\n", bucketName)
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucketName)

	err = minioClient.SetBucketPolicy(context.Background(), bucketName, policy)
	if err != nil {
		slog.Error("Error while setting bucket policy: %v", err)
		return nil, err
	}

	return &MinIO{
		Client: minioClient,
		Cnf:    cnf,
	}, nil
}

func (m *MinIO) Upload(fileName, filePath string) (string, error) {
	// Fayl kengaytmasini asosida content turini aniqlash
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	bucketName := m.Cnf.MINIO_BUCKET_NAME

	// Agar contentType topilmasa, default qilib binary faylni o'rnatamiz
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Faylni yuklash
	_, err := m.Client.FPutObject(context.Background(), bucketName, fileName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		slog.Error("Error while uploading %s to bucket %s: %v\n", fileName, bucketName, err)
		return "", err
	}

	domain := m.Cnf.MINIO_DOMAIN
	minioURL := fmt.Sprintf("%s/%s/%s", domain, bucketName, fileName)

	return minioURL, nil
}
