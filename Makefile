.PHONY: run build ui dev test clean docker-build docker-push docker-up docker-down lint

## Start the self-contained Control Plane + UI on http://localhost:8080
run: build
	./factory

## Build the Next.js UI bundle and embed it into ./factory
build: ui
	go build -buildvcs=false -o ./factory .

## Compile Next.js static UI and sync into internal/server/dist
ui:
	npm --prefix web-ui run build
	rm -rf internal/server/dist/*
	cp -R web-ui/out/. internal/server/dist/

## Run Go API (:8080) or Next.js hot-reload UI (:3000) during UI iteration
dev:
	go run -buildvcs=false .

## Run all Go unit, race, and state-machine tests
test:
	go test -count=1 -race -buildvcs=false ./...

## Lint Go code (requires golangci-lint)
lint:
	golangci-lint run ./internal/... ./cmd/... 2>/dev/null || echo "golangci-lint not installed, skipping"

clean:
	rm -f ./factory
	rm -rf web-ui/.next web-ui/out

## Docker targets for containerized development

## Build Docker image for local development (tag: agentic-factory:dev)
docker-build:
	docker build -t agentic-factory:dev -f Dockerfile .

## Build and tag Docker image for ghcr.io registry
docker-push: docker-build
	docker tag agentic-factory:dev ghcr.io/pratikdhanave/agentic-factory:latest
	@echo "✅ Image tagged. Push with: docker push ghcr.io/pratikdhanave/agentic-factory:latest"

## Start factory in Docker using docker-compose
docker-up:
	docker-compose up --build

## Stop and remove factory Docker containers
docker-down:
	docker-compose down

## Start factory in Docker background mode
docker-daemon:
	docker-compose up -d --build
	@echo "✅ Factory running. View logs with: docker-compose logs -f"

## View Docker container logs
docker-logs:
	docker-compose logs -f factory

## Run factory binary inside Docker container
docker-shell:
	docker-compose exec factory sh
