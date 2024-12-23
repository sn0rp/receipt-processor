# Build stage for Go application
FROM golang:1.21-alpine AS BUILDER

WORKDIR /app

COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only Go files and config
COPY *.go ./
COPY config.yaml ./
COPY migrations ./migrations/

RUN go build -o receipt-processor

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=BUILDER /app/receipt-processor .
COPY --from=BUILDER /app/migrations ./migrations/
COPY config.yaml ./

EXPOSE 8080
CMD ["./receipt-processor"] 