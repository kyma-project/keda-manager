# Build the manager binary
FROM --platform=$BUILDPLATFORM europe-docker.pkg.dev/kyma-project/prod/external/library/golang:1.24.5-alpine3.22 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --chown=65532:65532 --from=builder /app/manager .
COPY --chown=65532:65532 --from=builder /app/keda.yaml .
USER 65532:65532

ENTRYPOINT ["/manager"]
