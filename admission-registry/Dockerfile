# Build the webhook binary
FROM golang:1.24 AS builder

#RUN apt-get -y update && apt-get -y install upx

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY pkg/ pkg/
COPY cmd/ cmd/

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn,direct"

RUN go mod download && \
    go build -v -a -o admission-registry main.go && \
    go build -v -a -o tls-manager cmd/tls/main.go && \
    upx admission-registry tls-manager

FROM debian:stable-slim AS admission
WORKDIR /app

COPY --from=builder /workspace/admission-registry .
RUN chmod +x admission-registry
CMD [ "./admission-registry" ]


FROM debian:stable-slim AS tls
COPY --from=builder /workspace/tls-manager .
RUN chmod +x tls-manager
CMD [ "./tls-manager" ]