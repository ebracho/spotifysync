FROM golang:1.15.0
WORKDIR /app
COPY . .
RUN go build -o spotifysync ./cmd/spotifysync/main.go

FROM ubuntu:18.04
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /app
COPY ./static /app/static
COPY --from=0 /app/spotifysync /app/spotifysync
CMD ["/app/spotifysync"]
