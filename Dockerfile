# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ .

# Build frontend
RUN npm run build

# Stage 2: Build backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Copy frontend build artifacts from previous stage to cmd/server/web for embed
COPY --from=frontend-builder /app/web ./cmd/server/web

# Build Go binary (statically linked)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary and static assets
COPY --from=backend-builder /app/server .
COPY --from=backend-builder /app/web ./web

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server"]

