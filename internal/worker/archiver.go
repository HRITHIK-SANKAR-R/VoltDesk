package worker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"voltdesk/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

func InitArchiver() {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")

	if accountID == "" || accessKey == "" || secretKey == "" {
		log.Println("[Archiver] Skipping R2 init: Missing credentials")
		return
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Printf("[Archiver] Failed to init config: %v", err)
		return
	}

	s3Client = s3.NewFromConfig(cfg)
	log.Println("[Archiver] R2 Client initialized successfully")
}

func ArchiveOldConversations(queries *models.Queries) {
	if s3Client == nil {
		return
	}

	convIDs, err := queries.GetOldResolvedConversations()
	if err != nil {
		log.Printf("[Archiver] Failed to get old conversations: %v", err)
		return
	}

	if len(convIDs) == 0 {
		return
	}
	
	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "voltdesk-archive"
	}

	for _, convID := range convIDs {
		// 1. Fetch messages
		messages, err := queries.GetMessagesForArchiving(convID)
		if err != nil {
			log.Printf("[Archiver] Failed to get messages for %s: %v", convID, err)
			continue
		}

		if len(messages) == 0 {
			// No messages to archive, just delete
			_ = queries.DeleteConversationAndMessages(convID)
			continue
		}

		// 2. Serialize to JSON
		jsonData, err := json.Marshal(messages)
		if err != nil {
			log.Printf("[Archiver] Failed to marshal %s: %v", convID, err)
			continue
		}

		// 3. Compress using GZIP
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(jsonData); err != nil {
			log.Printf("[Archiver] Failed to compress %s: %v", convID, err)
			continue
		}
		if err := gz.Close(); err != nil {
			log.Printf("[Archiver] Failed to close gzip %s: %v", convID, err)
			continue
		}

		// 4. Upload to Cloudflare R2
		key := fmt.Sprintf("archives/%s.json.gz", convID)
		_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(key),
			Body:        bytes.NewReader(buf.Bytes()),
			ContentType: aws.String("application/gzip"),
		})
		if err != nil {
			log.Printf("[Archiver] Failed to upload %s to R2: %v", convID, err)
			continue
		}

		// 5. Delete from Postgres only after 200 OK equivalent
		err = queries.DeleteConversationAndMessages(convID)
		if err != nil {
			log.Printf("[Archiver] Failed to delete %s from Postgres: %v", convID, err)
		} else {
			log.Printf("[Archiver] Successfully archived and deleted conversation: %s", convID)
		}
	}
}
