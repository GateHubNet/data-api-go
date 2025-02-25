GOPATH:=$(shell go env GOPATH)

.PHONY: init
# init tools-chain
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/favadi/protoc-go-inject-tag@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

.PHONY: proto
# generate proto files
proto:
	rm -rf pb doc/swagger && \
	mkdir -p internal/pb && mkdir -p doc/swagger && \
	protoc --proto_path=proto --go_out=internal/pb --go_opt=paths=source_relative \
	--go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=internal/pb --grpc-gateway_opt=paths=source_relative \
	--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=gatehub-data-api \
	--experimental_allow_proto3_optional \
    proto/*.proto \
	&& \
	protoc-go-inject-tag -input="./internal/pb/*.pb.go"

.PHONY: start
start:
	go run cmd/grpc/main.go & \
	go run cmd/http/main.go & \
	wait
