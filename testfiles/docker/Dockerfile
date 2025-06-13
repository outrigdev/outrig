FROM golang:1.24-alpine

WORKDIR /app

# Install Task (taskfile.dev)
RUN apk add --no-cache curl
RUN curl -sL https://taskfile.dev/install.sh | sh -s -- -b /usr/local/bin

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["task", "testsmall"]
