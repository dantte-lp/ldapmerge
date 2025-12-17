# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w \
        -X 'ldapmerge/internal/version.Version=${VERSION}' \
        -X 'ldapmerge/internal/version.Commit=${COMMIT}' \
        -X 'ldapmerge/internal/version.BuildDate=${BUILD_DATE}'" \
    -o /ldapmerge ./cmd/ldapmerge

# Final stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 ldapmerge && \
    adduser -u 1000 -G ldapmerge -s /bin/sh -D ldapmerge

WORKDIR /app

# Copy binary
COPY --from=builder /ldapmerge /usr/local/bin/ldapmerge

# Create data directory
RUN mkdir -p /data && chown ldapmerge:ldapmerge /data

USER ldapmerge

# Default port for API server
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

ENTRYPOINT ["ldapmerge"]
CMD ["--help"]
