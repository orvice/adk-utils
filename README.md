# adk-utils

Utility packages for the [Google ADK (Agent Development Kit)](https://google.github.io/adk-docs/) Go SDK.

## Packages

### artifact/s3artifact

An S3-backed implementation of `artifact.Service` for storing and retrieving artifacts. Compatible with Amazon S3 and S3-compatible services (MinIO, Cloudflare R2, etc.).

#### Usage

```go
import (
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/orvice/adk-utils/artifact/s3artifact"
)

// Load AWS config
cfg, err := config.LoadDefaultConfig(ctx)

// Standard AWS S3
client := s3artifact.NewS3Client(cfg, "")
svc := s3artifact.NewService("my-bucket", client)

// S3-compatible service with custom endpoint
client := s3artifact.NewS3Client(cfg, "https://play.min.io")
svc := s3artifact.NewService("my-bucket", client)
```