# Stage 0: Build Tailwind CSS
FROM node:20-alpine AS css-builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --silent

COPY static/css/input.css ./static/css/input.css
COPY tailwind.config.js ./
RUN npm run build:css

# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o server ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/static ./static
COPY --from=css-builder /app/static/css/output.css ./static/css/output.css

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/login || exit 1

CMD ["./server"]
