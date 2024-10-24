BINARY_NAME=gosvm

.PHONY: build
build:
	rm -f build/${BINARY_NAME}
	go build -o build/${BINARY_NAME} ./...

.PHONY: clean
clean:
	go clean
	rm build/${BINARY_NAME}

.PHONY: dep
dep:
	go mod download
	go mod verify

.PHONY: gdb
gdb:
	rm -f build/${BINARY_NAME}_gdb
	go build -gcflags "-N -l" -o build/${BINARY_NAME}_gdb ./...
