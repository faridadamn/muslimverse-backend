FROM golang:1.21-alpine

WORKDIR /app

# Copy go mod and sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build aplikasi
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]