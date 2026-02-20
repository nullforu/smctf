ARG GO_VERSION=1.25.5

FROM golang:${GO_VERSION} AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

RUN mkdir -p /out/logs

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build --chown=nonroot:nonroot /out/logs /app/logs

ENV HTTP_ADDR=:8081 \
    LOG_DIR=/app/logs

EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
