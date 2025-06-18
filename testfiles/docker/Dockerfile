FROM golang:1.24-alpine

WORKDIR /app

# Install Task (taskfile.dev)
RUN apk add --no-cache curl
RUN curl -sL https://taskfile.dev/install.sh | sh -s -- -b /usr/local/bin

# ---- START: Outrig Integration ----
RUN apk add --no-cache curl
RUN curl -sf https://outrig.run/install.sh?x=1 | sh
# ---- END: Outrig Integration ----

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN rm -rf bin/

CMD ["task", "test:small"]
