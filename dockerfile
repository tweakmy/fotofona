# Start from golang v1.11 base image
FROM golang:1.12 AS builder

LABEL maintainer="JO EE <liewjoee@yahoo.com>"

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/github.com/tweakmy/fotofona

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN dep ensure

RUN make build

RUN echo $GOPATH

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/tweakmy/fotofona/bin/fotofona .
CMD ["./fotofona", "-h"]