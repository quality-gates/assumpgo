# Development image: docker build -f dev.Dockerfile -t assumpgo-dev . && docker run --rm -it -v "$PWD":/workspace assumpgo-dev
FROM golang:alpine
RUN apk add --no-cache git bash make gcc musl-dev
WORKDIR /workspace
COPY . .
CMD ["go", "test", "./..."]
