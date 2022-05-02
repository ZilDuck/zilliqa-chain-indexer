FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git gcc g++ make libc-dev pkgconfig curl

RUN adduser -D -u 1001 -g '' appuser
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .

RUN go env CGO_ENABLED
RUN go mod download -x
RUN go mod verify

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/cli         ./cmd/cli
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/indexerd    ./cmd/indexerd
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/metadata    ./cmd/metadata
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/assetServer ./cmd/assetServer

RUN chmod u+x /go/bin/*

FROM alpine:latest

WORKDIR /app

RUN mkdir /app/logs

COPY --from=builder /etc/passwd         /etc/passwd
COPY --from=builder /go/bin/cli         /app/cli
COPY --from=builder /go/bin/indexerd    /app/indexerd
COPY --from=builder /go/bin/metadata    /app/metadata
COPY --from=builder /go/bin/assetServer /app/assetServer

COPY ./config/mappings               /app/config/mappings