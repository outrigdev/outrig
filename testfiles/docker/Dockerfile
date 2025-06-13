FROM golang:1.24-alpine

WORKDIR /app

# Install Task (taskfile.dev)
RUN apk add --no-cache curl
RUN curl -sL https://taskfile.dev/install.sh | sh -s -- -b /usr/local/bin

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN rm -rf bin/

# Install outrig binary to /root/.local/bin/outrig
RUN mkdir -p /root/.local/bin
COPY --chmod=755 bin/outrig-linux /root/.local/bin/outrig

CMD ["task", "testsmall"]
