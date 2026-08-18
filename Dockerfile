# Builder
FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY api api
COPY cmd cmd
COPY pkg pkg
COPY internal internal
COPY config config
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -extldflags '-static'" -o ./bin/main ./cmd/main.go

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /src

COPY --from=builder /src/bin/ /src/bin/
COPY config/ /src/config/

EXPOSE 8080

ENTRYPOINT ["/src/bin/main", "-v", "-iii"]
