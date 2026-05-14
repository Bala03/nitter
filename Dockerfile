FROM golang:1.22-alpine AS builder

WORKDIR /src/nitter
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o nitter .

FROM alpine:latest
WORKDIR /src/
RUN apk --no-cache add ca-certificates
COPY --from=builder /src/nitter/nitter ./
COPY --from=builder /src/nitter/nitter.example.conf ./nitter.conf
COPY --from=builder /src/nitter/public ./public
COPY --from=builder /src/nitter/templates ./templates
EXPOSE 8080
RUN adduser -h /src/ -D -s /bin/sh nitter
USER nitter
CMD ["./nitter"]
