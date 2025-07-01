# Stage 1: The builder stage
FROM golang:alpine AS builder

RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
ARG CACHE_BUSTER
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server -v ./cmd/api

# Stage 2: The final, minimal stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]