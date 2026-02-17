.PHONY: run
run:
	go run ./cmd/ozmade

.PHONY: build
build:
	go build -o ./out/ozmade ./cmd/ozmade
