# # Build the chantico-aggregator binary
FROM golang:1.23 AS builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY chantico/ chantico/

# Build
RUN go build -o chantico-aggregator ./chantico/aggregator/main.go

FROM debian:12-slim
WORKDIR /
COPY --from=builder /workspace/chantico-aggregator /bin/

# ENTRYPOINT ["/chantico-aggregator"]
