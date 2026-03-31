package s3artifact

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"google.golang.org/genai"

	"google.golang.org/adk/artifact"
)

// mockS3Client implements S3Client for testing.
type mockS3Client struct {
	objects map[string]*mockObject // key -> object
}

type mockObject struct {
	data        []byte
	contentType string
}

func newMockS3Client() *mockS3Client {
	return &mockS3Client{objects: make(map[string]*mockObject)}
}

func (m *mockS3Client) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, _ := io.ReadAll(params.Body)
	ct := "application/octet-stream"
	if params.ContentType != nil {
		ct = *params.ContentType
	}
	m.objects[*params.Key] = &mockObject{data: data, contentType: ct}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	obj, ok := m.objects[*params.Key]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(bytes.NewReader(obj.data)),
		ContentType: aws.String(obj.contentType),
	}, nil
}

func (m *mockS3Client) HeadObject(_ context.Context, params *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	obj, ok := m.objects[*params.Key]
	if !ok {
		return nil, &types.NotFound{}
	}
	return &s3.HeadObjectOutput{
		ContentType: aws.String(obj.contentType),
	}, nil
}

func (m *mockS3Client) DeleteObject(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	delete(m.objects, *params.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObjects(_ context.Context, params *s3.DeleteObjectsInput, _ ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	for _, obj := range params.Delete.Objects {
		delete(m.objects, *obj.Key)
	}
	return &s3.DeleteObjectsOutput{}, nil
}

func (m *mockS3Client) ListObjectsV2(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var contents []types.Object
	prefix := ""
	if params.Prefix != nil {
		prefix = *params.Prefix
	}
	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			contents = append(contents, types.Object{Key: aws.String(key)})
		}
	}
	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

func TestSaveAndLoad(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	// Save text artifact
	saveResp, err := svc.Save(ctx, &artifact.SaveRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "test.txt",
		Part: &genai.Part{Text: "hello world"},
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saveResp.Version != 1 {
		t.Fatalf("expected version 1, got %d", saveResp.Version)
	}

	// Load it back
	loadResp, err := svc.Load(ctx, &artifact.LoadRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "test.txt",
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loadResp.Part.InlineData == nil {
		t.Fatal("expected InlineData in response")
	}
	if string(loadResp.Part.InlineData.Data) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(loadResp.Part.InlineData.Data))
	}
}

func TestVersioning(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	for i := 0; i < 3; i++ {
		_, err := svc.Save(ctx, &artifact.SaveRequest{
			AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "file.txt",
			Part: &genai.Part{Text: "v" + strings.Repeat("x", i)},
		})
		if err != nil {
			t.Fatalf("Save %d failed: %v", i, err)
		}
	}

	// Versions should return 3 versions
	versResp, err := svc.Versions(ctx, &artifact.VersionsRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "file.txt",
	})
	if err != nil {
		t.Fatalf("Versions failed: %v", err)
	}
	if len(versResp.Versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versResp.Versions))
	}

	// Load specific version
	loadResp, err := svc.Load(ctx, &artifact.LoadRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "file.txt", Version: 1,
	})
	if err != nil {
		t.Fatalf("Load v1 failed: %v", err)
	}
	if loadResp.Part.InlineData == nil || string(loadResp.Part.InlineData.Data) != "v" {
		t.Fatalf("expected 'v', got %v", loadResp.Part)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	for _, name := range []string{"a.txt", "b.txt"} {
		_, err := svc.Save(ctx, &artifact.SaveRequest{
			AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: name,
			Part: &genai.Part{Text: "data"},
		})
		if err != nil {
			t.Fatalf("Save %s failed: %v", name, err)
		}
	}

	listResp, err := svc.List(ctx, &artifact.ListRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listResp.FileNames) != 2 {
		t.Fatalf("expected 2 files, got %d", len(listResp.FileNames))
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	_, err := svc.Save(ctx, &artifact.SaveRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "del.txt",
		Part: &genai.Part{Text: "to delete"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(ctx, &artifact.DeleteRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "del.txt",
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.Versions(ctx, &artifact.VersionsRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "del.txt",
	})
	if err == nil {
		t.Fatal("expected error after deleting all versions")
	}
}

func TestUserScopedArtifact(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	_, err := svc.Save(ctx, &artifact.SaveRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "user:profile.json",
		Part: &genai.Part{Text: `{"name":"test"}`},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should appear in list
	listResp, err := svc.List(ctx, &artifact.ListRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1",
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, name := range listResp.FileNames {
		if name == "user:profile.json" {
			found = true
		}
	}
	if !found {
		t.Fatal("user-scoped artifact not found in list")
	}
}

func TestSaveInlineData(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	data := []byte{0x89, 0x50, 0x4e, 0x47} // PNG header
	_, err := svc.Save(ctx, &artifact.SaveRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "image.png",
		Part: &genai.Part{InlineData: &genai.Blob{MIMEType: "image/png", Data: data}},
	})
	if err != nil {
		t.Fatal(err)
	}

	loadResp, err := svc.Load(ctx, &artifact.LoadRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "image.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if loadResp.Part.InlineData == nil {
		t.Fatal("expected InlineData")
	}
	if !bytes.Equal(loadResp.Part.InlineData.Data, data) {
		t.Fatal("data mismatch")
	}
}

func TestLoadNotFound(t *testing.T) {
	ctx := context.Background()
	svc := NewService("test-bucket", newMockS3Client())

	_, err := svc.Load(ctx, &artifact.LoadRequest{
		AppName: "app1", UserID: "user1", SessionID: "sess1", FileName: "missing.txt",
	})
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}
