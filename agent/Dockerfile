FROM golang:1.22.0-alpine3.19 as build
WORKDIR /usr/src/app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /my-awesome-app

FROM alpine:3.19 as prod
COPY --from=build /my-awesome-app /my-awesome-app
ENTRYPOINT ["/my-awesome-app"]