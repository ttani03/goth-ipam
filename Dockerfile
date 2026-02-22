# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache nodejs npm git

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

COPY package*.json ./
RUN npm install

# Copy source code
COPY . .

# Generate templ files
RUN templ generate

# Build CSS
RUN npx @tailwindcss/cli -i ./global.css -o ./static/css/global.css

# Build the application
RUN CGO_ENABLED=0 go build -o /ipam ./cmd/ipam

# Final stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /ipam .
# Copy static files (needed for CSS etc.)
COPY --from=builder /app/static ./static
# Copy schema for initial setup
COPY --from=builder /app/internal/database/schema.sql ./internal/database/schema.sql

EXPOSE 8080

CMD ["./ipam"]
