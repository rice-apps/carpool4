# syntax=docker/dockerfile:1

# Adapted from https://docs.docker.com/guides/golang/

# Build the application from source
FROM golang:1.27.1 AS build-stage

WORKDIR /app

COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Deploy the application binary into a lean image
FROM gcr.io/distroless/static-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /server /server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/server"]
