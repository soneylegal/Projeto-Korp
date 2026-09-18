# --- build stage ---
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod ./
COPY main.go ./

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/http-server-projeto-korp .

# --- runtime stage ---
FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/http-server-projeto-korp /usr/local/bin/http-server-projeto-korp

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/http-server-projeto-korp"]
