all: build

build:
	CGO_ENABLED=1 go build -v -o bin/feedfusion .

test:
	go test ./...

tidy:
	go mod tidy

watch:
	modd

run:
	./bin/feedfusion

stop:
	pkill -f feedfusion || true
