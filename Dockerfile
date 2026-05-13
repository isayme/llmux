FROM golang:1.26-alpine AS go-builder
WORKDIR /app

COPY server .
# RUN mkdir -p ./dist && GO111MODULE=on GOPROXY=https://goproxy.cn,direct go mod download
RUN mkdir -p ./dist && GO111MODULE=on go mod download
RUN go build -o ./dist/llmux main.go

FROM node:24-slim AS node-builder
WORKDIR /app

COPY web .
RUN npm i -g pnpm@10
RUN CI=true pnpm i --frozen-lockfile
RUN pnpm build

FROM alpine
WORKDIR /app

COPY --from=go-builder /app/dist/llmux /app/llmux
COPY --from=node-builder /app/dist /app/public

CMD ["/app/llmux"]
