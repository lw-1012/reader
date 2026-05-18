# ---------- frontend build ----------
FROM node:20-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN mkdir -p ../backend/webui && npm run build

# ---------- backend build ----------
FROM golang:1.22-alpine AS go
WORKDIR /src
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum* ./
RUN go mod download 2>/dev/null || true
COPY backend/ ./
# inject built frontend into embed dir
COPY --from=web /backend/webui ./webui
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/reader .
# prepare /data owned by nonroot uid so named volumes inherit ownership
RUN mkdir -p /out/data && chown -R 65532:65532 /out/data

# ---------- runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=go /out/reader /app/reader
COPY --from=go --chown=65532:65532 /out/data /data
ENV READER_ADDR=":8080" \
    READER_DATA_DIR="/data"
EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/app/reader"]
