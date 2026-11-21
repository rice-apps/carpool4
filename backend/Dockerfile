FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=build /server /server
USER nobody
EXPOSE 8080
CMD ["/server"]
