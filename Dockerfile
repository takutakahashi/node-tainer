# Build the manager binary
FROM golang:1.19 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/
COPY cmd/ cmd/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o node-tainter main.go

FROM ubuntu
WORKDIR /
RUN apt update && apt install -y iputils-ping curl
COPY --from=builder /workspace/node-tainter .
USER 65532:65532

ENTRYPOINT ["/node-tainter"]
