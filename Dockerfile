FROM golang:1.23-alpine as builder

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o cmd/server ./cmd/main.go

# Step 2: Create the production image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/ .

# Expose the port the app runs on
EXPOSE 8080

# Start the app
CMD ["./cmd/server"]
