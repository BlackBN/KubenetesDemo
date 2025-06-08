FROM golang:1.24 AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY pkg/ pkg/

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn,direct"
RUN go mod download && go build -v -o admission-registry main.go

FROM debian:stable-slim
WORKDIR /app
COPY --from=builder /workspace/admission-registry .
RUN chmod +x admission-registry
CMD [ "./admission-registry" ]