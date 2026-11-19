FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev sqlite-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /asf ./

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

COPY --from=builder /asf /app/asf

RUN mkdir -p /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD /app/asf help || exit 1

ENTRYPOINT ["/app/asf"]
CMD []
