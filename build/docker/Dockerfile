# build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mcm ./cmd/mcm

# run stage
FROM gcr.io/distroless/static
COPY --from=build /out/mcm /mcm
EXPOSE 8080
ENTRYPOINT ["/mcm"]
