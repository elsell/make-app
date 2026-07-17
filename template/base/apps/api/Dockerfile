# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e
FROM golang:1.25.12-alpine@sha256:56961d79ea8129efddcc0b8643fd8a5416b4e6228cfd477e3fd61deb2672c587 AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go-build-cache GOCACHE=/tmp/go-build-cache CGO_ENABLED=0 go build -o /app ./cmd/server
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go-build-cache GOCACHE=/tmp/go-build-cache CGO_ENABLED=0 go build -o /schema ./cmd/schema
FROM gcr.io/distroless/static-debian12:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0
COPY --from=build /app /app
COPY --from=build /schema /schema
ENTRYPOINT ["/app"]
