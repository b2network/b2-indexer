FROM golang:latest as builder
COPY . /src
RUN apt-get update && apt install -y protobuf-compiler git && \
    cd /tmp && git clone https://github.com/googleapis/googleapis.git && \
    cp -r /tmp/googleapis/* /usr/local/include/ && \
    cd /src && \
    go get github.com/golang/protobuf/protoc-gen-go && \
    go install github.com/golang/protobuf/protoc-gen-go && \
    #    PROTO_INCLUDE=/usr/local/include make proto && \
    GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/main main.go

# =====
FROM alpine:latest
COPY --from=builder /src/build/main /usr/bin/main
# COPY --from=builder /src/api/protobuf/api.swagger.json /api.swagger.json
CMD ["/usr/bin/main","start"]

