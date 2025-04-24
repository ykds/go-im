services := cmd/gateway cmd/user cmd/message cmd/access cmd/seqserver
BUILD_DIR := build

ifeq ($(OS),Windows_NT)
	GOOS=windows
	SUFFIX=.exe
	KILL=taskkill /F /IM
else
	GOOS=linux
	SUFFIX=
	KILL=pkill -9
endif

.PHONY: build-all-windows build-all-linux build-gateway build-user build-message build-access build-seqserver clean

build-all-windows:
	@mkdir -p $(BUILD_DIR)
	@for service in $(services); do \
		echo "build $$service..."; \
		GOOS=windows go build -o ./build/$$(basename $$service)$(SUFFIX) ./$$service/main.go; \
	done

build-all-linux:
	@mkdir -p $(BUILD_DIR)
	@for service in $(services); do \
		echo "build $$service..."; \
		GOOS=linux go build -o ./build/$$(basename $$service) ./$$service/main.go; \
	done

build-gateway:
	@echo "building gateway..."
	GOOS=$(GOOS) go build -o ./build/gateway$(SUFFIX) ./cmd/gateway/main.go

build-user:
	@echo "building user..."
	GOOS=$(GOOS) go build -o ./build/user$(SUFFIX) ./cmd/user/main.go

build-message:
	@echo "building message..."
	GOOS=$(GOOS) go build -o ./build/message$(SUFFIX) ./cmd/message/main.go

build-access:
	@echo "building access..."
	GOOS=$(GOOS) go build -o ./build/access$(SUFFIX) ./cmd/access/main.go

build-seqserver:
	@echo "building seqserver..."
	GOOS=$(GOOS) go build -o ./build/seqserver$(SUFFIX) ./cmd/seqserver/main.go
	
clean:
	@echo "cleanning..."
	@rm -rf $(BUILD_DIR)


.PHONY: user-api, message-api, access-api
user-api:
	protoc --go_opt=paths=source_relative --go_out=. --go-grpc_opt=paths=source_relative --go-grpc_out=. api/user/user.proto

message-api:
	protoc --go_opt=paths=source_relative --go_out=. --go-grpc_opt=paths=source_relative --go-grpc_out=. api/message/message.proto
