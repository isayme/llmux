.PHONY: buildweb
buildweb:
	mkdir -p server/dist && cd web && npm run build && cp -r ./dist/* ../server/dist

.PHONY: dev
dev: buildweb
	cd server && go run main.go

.PHONY: test-convert
test-convert:
	cd server && go test -cover -coverprofile=coverage.out ./internal/handler/convert/ && go tool cover -func=coverage.out
