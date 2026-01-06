FROM golang:1.25.4-alpine AS builder

WORKDIR /gomigrator

COPY . .
RUN go mod download && go mod verify

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gomigrator ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /gomigrator/gomigrator /usr/local/bin/gomigrator

CMD ["gomigrator"]
