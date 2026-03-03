# Multi-stage build for nickai CLI and node binaries.
# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/nickai .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/nickai-node ./cmd/node/

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /home/nickai nickai

USER nickai
WORKDIR /home/nickai

COPY --from=builder /out/nickai /usr/local/bin/nickai
COPY --from=builder /out/nickai-node /usr/local/bin/nickai-node

VOLUME /home/nickai/.nickai

EXPOSE 9400

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD nickai-node ping localhost:9400 || exit 1

ENTRYPOINT ["nickai-node"]
