# Runtime image: docker build -t assumpgo . && docker run --rm -v "$PWD":/code assumpgo ./...
FROM golang:alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/assumpgo ./cmd/assumpgo

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /code
COPY --from=build /app/assumpgo /usr/local/bin/assumpgo
ENTRYPOINT ["assumpgo"]
CMD ["./..."]
