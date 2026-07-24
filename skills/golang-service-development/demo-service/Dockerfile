# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically; CGO disabled to keep image small.
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/server ./cmd/server/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/server /server
USER nonroot:nonroot
EXPOSE 9000 8080
ENTRYPOINT ["/server"]
