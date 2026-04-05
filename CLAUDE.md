# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go utility packages for the [Google ADK (Agent Development Kit)](https://google.github.io/adk-docs/) Go SDK (`google.golang.org/adk`). Provides alternative backend implementations for ADK interfaces.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./artifact/s3artifact/

# Run a single test
go test ./artifact/s3artifact/ -run TestSaveAndLoad
```

## Architecture

The repo provides pluggable implementations of ADK interfaces. Each package lives under a directory matching the ADK interface it implements.

- **`artifact/s3artifact`** — Implements `artifact.Service` from `google.golang.org/adk/artifact` using Amazon S3. Supports S3-compatible services (MinIO, R2) via custom endpoints. Uses an `S3Client` interface (subset of `*s3.Client`) for testability — tests use an in-memory mock rather than a real S3 backend.

### S3 artifact key layout

Artifacts are stored with versioned keys:
- Session-scoped: `{appName}/{userID}/{sessionID}/{fileName}/{version}`
- User-scoped (filename prefixed with `user:`): `{appName}/{userID}/user/{fileName}/{version}`
