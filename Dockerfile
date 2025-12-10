# ---- Build stage ---------------------------------------------------------
FROM golang:1.25 AS builder

# Turn on modules (usually default, but explicit is fine)
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

# Cache deps first
COPY go.mod ./
RUN go mod download

# Copy the rest of the source (including embedded static files)
COPY . .

# Build a static binary
RUN go build -o /jultelegrafen ./...

FROM alpine:3.23.0

RUN apk add --no-cache ca-certificates

# Copy the statically linked binary from builder
COPY --from=builder /jultelegrafen /jultelegrafen
COPY --from=builder /app/messages.json /messages.json

# Container listens on same port as main.go
EXPOSE 8080

# Run the server
ENTRYPOINT ["/jultelegrafen"]
