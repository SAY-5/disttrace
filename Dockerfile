FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go test ./... && go build -o /disttrace ./cmd/disttrace

FROM alpine:3.20
RUN adduser -D -u 1001 dt
USER dt
COPY --from=build /disttrace /usr/local/bin/disttrace
EXPOSE 8080
ENTRYPOINT ["disttrace", "-addr", "0.0.0.0:8080"]
