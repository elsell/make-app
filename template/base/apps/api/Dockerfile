# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e
FROM golang:1.25.12-alpine@sha256:56961d79ea8129efddcc0b8643fd8a5416b4e6228cfd477e3fd61deb2672c587 AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api .
RUN CGO_ENABLED=0 go build -o /app ./cmd/server
RUN CGO_ENABLED=0 go build -o /schema ./cmd/schema
RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate
RUN CGO_ENABLED=0 go build -o /spicedb-runtime-proxy ./cmd/spicedb-runtime-proxy
FROM gcr.io/distroless/static-debian12:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0
COPY --from=build /app /app
COPY --from=build /schema /schema
COPY --from=build /migrate /migrate
COPY --from=build /spicedb-runtime-proxy /spicedb-runtime-proxy
ENTRYPOINT ["/app"]
