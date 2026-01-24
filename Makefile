.PHONY: run build clean

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
