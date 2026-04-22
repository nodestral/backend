build:
	CGO_ENABLED=1 go build -o bin/nodestral-backend ./cmd/server

run: build
	./bin/nodestral-backend

test:
	go test ./...

clean:
	rm -rf bin/ nodestral.db
