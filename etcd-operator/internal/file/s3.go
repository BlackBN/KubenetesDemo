package file

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Uploader struct {
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Secure          bool
}

func NewS3Uploader(ep, ak, sk string, useSSL bool) *s3Uploader {
	return &s3Uploader{
		Endpoint:        ep,
		AccessKeyId:     ak,
		SecretAccessKey: sk,
		Secure:          useSSL,
	}
}

func (su *s3Uploader) InitClient() (*minio.Client, error) {
	// Initialize minio client object.
	return minio.New(su.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(su.AccessKeyId, su.SecretAccessKey, ""),
		Secure: su.Secure,
	})
}

func (su *s3Uploader) Upload(ctx context.Context, filePath string) (int64, error) {
	minioClient, err := su.InitClient()
	if err != nil {
		return 0, err
	}
	backetName := "bn"
	objectName := "etcd-snapshot.db"
	uploadInfo, err := minioClient.FPutObject(ctx, backetName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}
	return uploadInfo.Size, nil
}
