// Package s3artifact provides an Amazon S3 [artifact.Service].
//
// This package allows storing and retrieving artifacts in an S3 bucket.
// Artifacts are organized by application name, user ID, session ID, and filename,
// with support for versioning.
package s3artifact

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"google.golang.org/genai"

	"google.golang.org/adk/artifact"
)

// s3Service is an Amazon S3 implementation of the artifact.Service.
type s3Service struct {
	bucketName string
	client     S3Client
}

// S3Client is the subset of the S3 API used by this package. It can be satisfied
// by *s3.Client or a mock for testing.
type S3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// NewService creates a new S3-backed artifact service for the specified bucket.
func NewService(bucketName string, client S3Client) artifact.Service {
	return &s3Service{
		bucketName: bucketName,
		client:     client,
	}
}

// NewS3Client creates an S3 client from the given AWS config.
// If endpoint is non-empty, the client will use it as the S3 endpoint URL,
// which is useful for S3-compatible services such as MinIO or Cloudflare R2.
func NewS3Client(cfg aws.Config, endpoint string) *s3.Client {
	if endpoint != "" {
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	return s3.NewFromConfig(cfg)
}

func fileHasUserNamespace(filename string) bool {
	return strings.HasPrefix(filename, "user:")
}

func buildBlobName(appName, userID, sessionID, fileName string, version int64) string {
	if fileHasUserNamespace(fileName) {
		return fmt.Sprintf("%s/%s/user/%s/%d", appName, userID, fileName, version)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%d", appName, userID, sessionID, fileName, version)
}

func buildBlobNamePrefix(appName, userID, sessionID, fileName string) string {
	if fileHasUserNamespace(fileName) {
		return fmt.Sprintf("%s/%s/user/%s/", appName, userID, fileName)
	}
	return fmt.Sprintf("%s/%s/%s/%s/", appName, userID, sessionID, fileName)
}

func buildSessionPrefix(appName, userID, sessionID string) string {
	return fmt.Sprintf("%s/%s/%s/", appName, userID, sessionID)
}

func buildUserPrefix(appName, userID string) string {
	return fmt.Sprintf("%s/%s/user/", appName, userID)
}

// Save implements [artifact.Service].
func (s *s3Service) Save(ctx context.Context, req *artifact.SaveRequest) (*artifact.SaveResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	nextVersion := int64(1)
	response, err := s.versions(ctx, &artifact.VersionsRequest{
		AppName: req.AppName, UserID: req.UserID, SessionID: req.SessionID, FileName: req.FileName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact versions: %w", err)
	}
	if len(response.Versions) > 0 {
		nextVersion = slices.Max(response.Versions) + 1
	}

	key := buildBlobName(req.AppName, req.UserID, req.SessionID, req.FileName, nextVersion)

	var body []byte
	var contentType string
	if req.Part.InlineData != nil {
		body = req.Part.InlineData.Data
		contentType = req.Part.InlineData.MIMEType
	} else {
		body = []byte(req.Part.Text)
		contentType = "text/plain"
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write object to S3: %w", err)
	}

	return &artifact.SaveResponse{Version: nextVersion}, nil
}

// Load implements [artifact.Service].
func (s *s3Service) Load(ctx context.Context, req *artifact.LoadRequest) (*artifact.LoadResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	version := req.Version
	if version == 0 {
		response, err := s.versions(ctx, &artifact.VersionsRequest{
			AppName: req.AppName, UserID: req.UserID, SessionID: req.SessionID, FileName: req.FileName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list artifact versions: %w", err)
		}
		if len(response.Versions) == 0 {
			return nil, fmt.Errorf("artifact not found: %w", fs.ErrNotExist)
		}
		version = slices.Max(response.Versions)
	}

	key := buildBlobName(req.AppName, req.UserID, req.SessionID, req.FileName, version)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("artifact '%s' not found: %w", key, fs.ErrNotExist)
		}
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	ct := "application/octet-stream"
	if result.ContentType != nil {
		ct = *result.ContentType
	}

	part := genai.NewPartFromBytes(data, ct)
	return &artifact.LoadResponse{Part: part}, nil
}

// Delete implements [artifact.Service].
func (s *s3Service) Delete(ctx context.Context, req *artifact.DeleteRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("request validation failed: %w", err)
	}

	// Delete specific version
	if req.Version != 0 {
		key := buildBlobName(req.AppName, req.UserID, req.SessionID, req.FileName, req.Version)
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("failed to delete artifact: %w", err)
		}
		return nil
	}

	// Delete all versions
	response, err := s.versions(ctx, &artifact.VersionsRequest{
		AppName: req.AppName, UserID: req.UserID, SessionID: req.SessionID, FileName: req.FileName,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch versions on delete: %w", err)
	}
	if len(response.Versions) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(response.Versions))
	for i, v := range response.Versions {
		key := buildBlobName(req.AppName, req.UserID, req.SessionID, req.FileName, v)
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("failed to batch delete artifacts: %w", err)
	}
	return nil
}

// List implements [artifact.Service].
func (s *s3Service) List(ctx context.Context, req *artifact.ListRequest) (*artifact.ListResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	filenamesSet := map[string]bool{}

	// Fetch session-scoped filenames.
	if err := s.fetchFilenamesFromPrefix(ctx, buildSessionPrefix(req.AppName, req.UserID, req.SessionID), filenamesSet); err != nil {
		return nil, fmt.Errorf("failed to fetch session filenames: %w", err)
	}

	// Fetch user-scoped filenames.
	if err := s.fetchFilenamesFromPrefix(ctx, buildUserPrefix(req.AppName, req.UserID), filenamesSet); err != nil {
		return nil, fmt.Errorf("failed to fetch user filenames: %w", err)
	}

	filenames := make([]string, 0, len(filenamesSet))
	for name := range filenamesSet {
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)
	return &artifact.ListResponse{FileNames: filenames}, nil
}

// Versions implements [artifact.Service] and returns an error if no versions are found.
func (s *s3Service) Versions(ctx context.Context, req *artifact.VersionsRequest) (*artifact.VersionsResponse, error) {
	response, err := s.versions(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(response.Versions) == 0 {
		return nil, fmt.Errorf("artifact not found: %w", fs.ErrNotExist)
	}
	return response, nil
}

// versions is the internal version that does not error on empty results.
func (s *s3Service) versions(ctx context.Context, req *artifact.VersionsRequest) (*artifact.VersionsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	prefix := buildBlobNamePrefix(req.AppName, req.UserID, req.SessionID, req.FileName)
	versions := make([]int64, 0)

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			segments := strings.Split(*obj.Key, "/")
			if len(segments) < 1 {
				continue
			}
			v, err := strconv.ParseInt(segments[len(segments)-1], 10, 64)
			if err != nil {
				continue
			}
			versions = append(versions, v)
		}
	}

	return &artifact.VersionsResponse{Versions: versions}, nil
}

func (s *s3Service) fetchFilenamesFromPrefix(ctx context.Context, prefix string, filenamesSet map[string]bool) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			segments := strings.Split(*obj.Key, "/")
			if len(segments) < 2 {
				return fmt.Errorf("incorrect number of segments in path %q", *obj.Key)
			}
			filename := segments[len(segments)-2]
			filenamesSet[filename] = true
		}
	}
	return nil
}

// isNotFound checks if an error is an S3 "not found" error.
func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if ok := errors.As(err, &nsk); ok {
		return true
	}
	var nf *types.NotFound
	return errors.As(err, &nf)
}
