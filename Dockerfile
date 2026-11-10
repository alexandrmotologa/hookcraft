# Multi-stage build for HookCraft (< 15MB final image)
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/bin/hookcraft ./cmd/hookcraft

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates
COPY --from=go-builder /app/bin/hookcraft /usr/local/bin/hookcraft
ENTRYPOINT ["hookcraft"]
CMD ["help"]
