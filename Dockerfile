FROM --platform=$BUILDPLATFORM golang:1.21.2 AS build

WORKDIR /build

COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app
RUN apt-get update && apt-get install -y ca-certificates

FROM --platform=$BUILDPLATFORM scratch

WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app /app

EXPOSE 8080

ENTRYPOINT [ "/app" ]