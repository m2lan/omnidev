// Package builder provides Docker image building capabilities.
package builder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

// BuildResult represents the result of a build.
type BuildResult struct {
	ImageTag  string `json:"image_tag"`
	BuildLogs string `json:"build_logs"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Duration  int    `json:"duration_ms"`
}

// Builder defines the interface for image building.
type Builder interface {
	// Build builds a Docker image from a project.
	Build(ctx context.Context, opts BuildOptions) (*BuildResult, error)

	// DetectFramework detects the project framework and generates a Dockerfile.
	DetectFramework(projectPath string) (string, error)
}

// BuildOptions contains build configuration.
type BuildOptions struct {
	ProjectPath  string
	ImageName    string
	ImageTag     string
	Dockerfile   string
	BuildCommand string
	BuildArgs    map[string]string
}

// DockerBuilder builds Docker images.
type DockerBuilder struct{}

// NewDockerBuilder creates a new Docker builder.
func NewDockerBuilder() *DockerBuilder {
	return &DockerBuilder{}
}

// Build builds a Docker image.
func (b *DockerBuilder) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	start := time.Now()

	// Generate image tag if not provided
	if opts.ImageTag == "" {
		opts.ImageTag = fmt.Sprintf("%s:%s", opts.ImageName, time.Now().Format("20060102-150405"))
	}

	// Build args
	args := []string{"build", "-t", opts.ImageTag}
	for k, v := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}
	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}
	args = append(args, opts.ProjectPath)

	logger.Log.Info("Building Docker image",
		zap.String("image", opts.ImageTag),
		zap.String("path", opts.ProjectPath),
	)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	latency := time.Since(start).Milliseconds()

	result := &BuildResult{
		ImageTag:  opts.ImageTag,
		BuildLogs: string(output),
		Duration:  int(latency),
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		logger.Log.Error("Build failed", zap.Error(err), zap.String("logs", string(output)))
		return result, fmt.Errorf("build failed: %w", err)
	}

	result.Success = true
	logger.Log.Info("Build succeeded", zap.String("image", opts.ImageTag), zap.Int64("duration_ms", latency))
	return result, nil
}

// DetectFramework detects the project framework and generates a Dockerfile.
func (b *DockerBuilder) DetectFramework(projectPath string) (string, error) {
	// Check for existing Dockerfile
	if fileExists(projectPath + "/Dockerfile") {
		return "Dockerfile", nil
	}

	// Detect framework
	if fileExists(projectPath + "/package.json") {
		if fileExists(projectPath + "/next.config.js") || fileExists(projectPath + "/next.config.ts") {
			return b.generateNextDockerfile(), nil
		}
		if fileExists(projectPath + "/vite.config.js") || fileExists(projectPath + "/vite.config.ts") {
			return b.generateViteDockerfile(), nil
		}
		return b.generateNodeDockerfile(), nil
	}

	if fileExists(projectPath + "/go.mod") {
		return b.generateGoDockerfile(), nil
	}

	if fileExists(projectPath + "/requirements.txt") || fileExists(projectPath + "/pyproject.toml") {
		return b.generatePythonDockerfile(), nil
	}

	if fileExists(projectPath + "/Cargo.toml") {
		return b.generateRustDockerfile(), nil
	}

	if fileExists(projectPath + "/pom.xml") || fileExists(projectPath + "/build.gradle") {
		return b.generateJavaDockerfile(), nil
	}

	return "", fmt.Errorf("unable to detect project framework")
}

func (b *DockerBuilder) generateGoDockerfile() string {
	return `FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /app/server /app/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server"]`
}

func (b *DockerBuilder) generateNodeDockerfile() string {
	return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 3000
CMD ["npm", "start"]`
}

func (b *DockerBuilder) generateNextDockerfile() string {
	return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 3000
CMD ["node", "server.js"]`
}

func (b *DockerBuilder) generateViteDockerfile() string {
	return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`
}

func (b *DockerBuilder) generatePythonDockerfile() string {
	return `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN adduser -D -g '' appuser
USER appuser
EXPOSE 8000
CMD ["python", "main.py"]`
}

func (b *DockerBuilder) generateRustDockerfile() string {
	return `FROM rust:1.77 AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
RUN adduser --disabled-password --gecos '' appuser
COPY --from=builder /app/target/release/server /app/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/server"]`
}

func (b *DockerBuilder) generateJavaDockerfile() string {
	return `FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package -DskipTests

FROM eclipse-temurin:21-jre
COPY --from=builder /app/target/*.jar /app/app.jar
RUN adduser --disabled-password --gecos '' appuser
USER appuser
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/app.jar"]`
}

func fileExists(path string) bool {
	_, err := exec.Command("test", "-f", path).Output()
	return err == nil
}

// isSimpleCheck does a simple file existence check using strings
func isSimpleCheck(path string) bool {
	return strings.Contains(path, ".")
}
