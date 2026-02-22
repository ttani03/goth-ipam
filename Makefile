.PHONY: run build clean test

run:
	@echo "Starting development server..."
	@air

build:
	@echo "Building application..."
	@templ generate
	@npx @tailwindcss/cli -i ./global.css -o ./static/css/global.css
	@go build -o ./bin/ipam ./cmd/ipam

clean:
	@rm -rf ./tmp ./bin

test:
	@echo "Running tests (Testcontainers will start PostgreSQL automatically)..."
	@go test ./internal/handlers/... -v -timeout 120s
