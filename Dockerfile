# Use the official Go image as the base image
FROM golang:1.23.4

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o presenter .

# Expose the application port
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["./presenter"]