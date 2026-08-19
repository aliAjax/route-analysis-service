FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/route-service ./cmd/route-service
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/route-service /route-service
EXPOSE 8084
USER nonroot:nonroot
ENTRYPOINT ["/route-service"]
