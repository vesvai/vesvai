vet:
	go vet ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test -v ./...

build:
	go build -o bin/vesvai cmd/vesvai/main.go

run:
	./bin/vesvai

clean:
	rm -rf bin