package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Kento75/s3-unzip-go/s3"
	"github.com/Kento75/s3-unzip-go/zip"
	"github.com/aws/aws-lambda-go/aws"
	"github.com/aws/aws-lambda-go/aws/endpoints"
	"github.com/aws/aws-lambda-go/aws/session"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

const (
	tempArtifactPath = "/tmp/artifact/"
	tempZipPath      = tempArtifactPath + "zipped/"
	tempUnzipPath    = tempArtifactPath + "unzipped/"
	tempZip          = "temp.zip"
	dirPerm          = 0777
	region           = endpoints.ApNortheast1RegionID
)

var (
	now string
	// zipファイルをダウンロードするlambda上のパス
	zipContentPath string
	// zipファイルを解凍するlambda上のパス
	unzipContentPath string
	// 解凍したファイルをアップロードするs3上のバケット
	destBucket string
)

func init() {
	destBucket = os.getenv("UNZIPPED_ARTIFACT_BUCKET")
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	if lc, ok = lambdacontext.FromContext(ctx); ok {
		log.Printf("AwsRequestID: %s", lc.AwsRequestID)
	}

	// バケット名取得
	bucket := s3Event.Records[0].S3.Bucket.Name
	// キーの取得
	key := s3Event.Records[0].S3.Object.Key

	log.Printf("bucket: %s, key: %s", bucket, key)

	// 空きスペースが確保できるか確認
	if err := prepareDirectory(); err != nil {
		log.Fatal(err)
	}

	// AWS接続情報の初期化
	// 初期化しない場合、~/.aws/credentialsが利用されるため
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region)
	}))

	downloader := s3.NewDownloader(sess, bucket, key, zipContentPath + tempZip)
	downloadedZipPath, err := downloader.Download()

	// ダウンロード失敗時
	if err != nil {
		log.Fatal(err)
	}

	// 解凍失敗時
	if err := zip.Unzip(downloadedZipPath, unzipContentPath); err != nil {
		log.Fatal(err)
	}

	uploader := s3.NewUploader(sess, tempUnzipPath, destBucket)

	// アップロードできなかった場合
	if err := uploader.Upload(); err != nil {
		log.Fatal(err)
	}

	log.Printf("%s unzipped to S3 bucket: %s", downloadedZipPath, destBucket)

	return nil
}

// スペース確保確認関数
func prepareDirectory() error {
	now = strconv.Itoa(int(time.Now().UnixNano()))
	zipContentPath = tempZipPath + now + "/"
	unzipContentPath = tempUnzipPath + now + "/"

	if _, err := os.Stat(tempArtifactPath); err != nil {
		if err := os.RemoveAll(tempArtifactPath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(zipContentPath, dirPerm); err != nil {
		return err
	}

	if err := os.MkdirAll(unzipContentPath, dirPerm); err != nil {
		return err
	}

	return nil
}
