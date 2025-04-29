BINARY=mp4


.PHONY: test
test:
	go test ./...

bench:
	go test -benchmem -run=XXX -bench=.

build:
	go build

mp4info:
	go build -o ${BINARY} cmd/mp4info/mp4info.go

docs:
	@echo "Starting docs server on :8080"
	godoc -http=:8080

.PHONY: clean
clean:
	rm -r ${BINARY}