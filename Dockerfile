FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ffmpeg yt-dlp git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata ffmpeg yt-dlp
WORKDIR /root/
COPY --from=builder /app/bot .
RUN mkdir -p /data /config
CMD ["./bot"]