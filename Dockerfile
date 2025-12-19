# Multi-stage build for Cloud Run
# Stage 1: build
FROM golang:1.24 as builder
WORKDIR /app

# Disable CGO for static binary (modernc sqlite is pure Go)
ENV CGO_ENABLED=0
ENV GOOS=linux

# Pre-fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build backend server
RUN go build -o server ./cmd/api

# Stage 2: minimal runtime image
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app



# Copy binary
COPY --from=builder /app/server /app/server

# Run as nonroot (distroless provides nonroot user)
USER nonroot:nonroot

EXPOSE 8080

CMD ["/app/server"]
