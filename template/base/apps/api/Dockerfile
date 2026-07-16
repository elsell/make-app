# syntax=docker/dockerfile:1.7
FROM registry.access.redhat.com/hi/go:1.25.10-builder-1780418048@sha256:1a99d42f555db97455998945faf3c797c1f65ce1b92e4d9952a589446d114d6c AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go-build-cache GOCACHE=/tmp/go-build-cache CGO_ENABLED=0 go build -o /app ./cmd/server
FROM gcr.io/distroless/static-debian12:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0
COPY --from=build /app /app
ENTRYPOINT ["/app"]
