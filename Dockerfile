FROM golang:1.15 as builder

# All these steps will be cached
RUN mkdir /build
WORKDIR /build
# <- COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .

# Get dependencies - will also be cached if we won't change mod/sum
RUN go mod download

COPY . .

RUN make vendor build