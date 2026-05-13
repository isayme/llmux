.PHONY: buildweb
buildweb:
	mkdir -p server/dist && cd web && npm run build && cp -r ./dist/* ../server/dist

.PHONY: dev
dev: buildweb
	cd server && go run main.go
