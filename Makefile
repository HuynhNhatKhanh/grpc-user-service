.PHONY: proto clean-proto install-tools

# Install protoc plugins
install-tools:
	go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
	go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
	go get -tool google.golang.org/protobuf/cmd/protoc-gen-go
	go get -tool google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Download googleapis if not exists
third_party/googleapis:
	mkdir -p third_party
	git clone --depth 1 https://github.com/googleapis/googleapis.git third_party/googleapis

# Generate protobuf and gRPC code
proto: third_party/googleapis
	protoc -I . -I third_party/googleapis \
		--go_out=. --go_opt=module=grpc-user-service \
		--go-grpc_out=. --go-grpc_opt=module=grpc-user-service \
		--grpc-gateway_out=. --grpc-gateway_opt=module=grpc-user-service \
		--grpc-gateway_opt=generate_unbound_methods=true \
		api/proto/user.proto

# Clean generated files
clean-proto:
	rm -f api/gen/go/user/*.pb.go
	rm -f api/gen/go/user/*.pb.gw.go

# Regenerate all proto files
regen-proto: clean-proto proto

# Code quality commands
lint: 
	golangci-lint run ./...

# Code formatter
format:
	gofmt -s -w .
