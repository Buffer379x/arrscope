# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM node:24.19.0-alpine AS frontend
RUN npm install -g pnpm@11.25.0
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=0.1.0-dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /arrscope ./cmd/arrscope

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && mkdir /data && chown 99:100 /data
COPY --from=backend /arrscope /usr/local/bin/arrscope
ARG VERSION=0.1.0-dev
ARG SOURCE=https://github.com/Buffer379x/arrscope
LABEL org.opencontainers.image.title="ArrScope" org.opencontainers.image.description="Filter your Radarr and Sonarr library without scanning media" org.opencontainers.image.source=$SOURCE org.opencontainers.image.version=$VERSION org.opencontainers.image.licenses="MIT"
ENV ARRSCOPE_DATA_DIR=/data ARRSCOPE_LISTEN=:8080 GOMEMLIMIT=192MiB GOMAXPROCS=2 SQLITE_TMPDIR=/data
USER 99:100
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["arrscope", "healthcheck"]
ENTRYPOINT ["arrscope"]
