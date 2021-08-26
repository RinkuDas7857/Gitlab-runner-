package s3

import (
	"context"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

const DefaultAWSS3Server = "s3.amazonaws.com"

type minioClient interface {
	PresignedGetObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		expires time.Duration,
		reqParams url.Values,
	) (*url.URL, error)
	PresignedPutObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		expires time.Duration,
	) (*url.URL, error)
}

var newMinio = minio.New
var newMinioWithIAM = func(serverAddress, bucketLocation string) (*minio.Client, error) {
	return minio.New(serverAddress, &minio.Options{
		Creds:  credentials.NewIAM(""),
		Secure: true,
		Transport: &bucketLocationTripper{
			bucketLocation: bucketLocation,
		},
	})
}

var newMinioClient = func(s3 *common.CacheS3Config) (minioClient, error) {
	serverAddress := s3.ServerAddress

	if serverAddress == "" {
		serverAddress = DefaultAWSS3Server
	}

	if s3.ShouldUseIAMCredentials() {
		return newMinioWithIAM(serverAddress, s3.BucketLocation)
	}

	return newMinio(serverAddress, &minio.Options{
		Creds:  credentials.NewStaticV4(s3.AccessKey, s3.SecretKey, ""),
		Secure: !s3.Insecure,
		Transport: &bucketLocationTripper{
			bucketLocation: s3.BucketLocation,
		},
	})
}
