FROM golang:1.24-alpine

WORKDIR /app

# Install Task (taskfile.dev)
RUN apk add --no-cache curl
RUN curl -sL https://taskfile.dev/install.sh | sh -s -- -b /usr/local/bin

# ---- START: Outrig Integration ----
ENV OUTRIG_VERSION=v0.9.0-beta.0
RUN apk add --no-cache curl
RUN curl -sf https://outrig.run/install.sh?x=1 | sh
RUN ln -s /root/.local/bin/outrig /usr/local/bin/outrig
# ---- END: Outrig Integration ----

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN rm -rf bin/

CMD ["task", "test:docker:small"]
