# syntax=docker/dockerfile:1
# Multi-stage image for AFI Go services.
# Build one binary per image via --build-arg AFI_SERVICE=<controlplane|gateway|worker|cli>

ARG GO_VERSION=1.25.0

FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG AFI_SERVICE=controlplane
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
	-o /out/afi-service ./cmd/${AFI_SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/afi-service /app/afi-service
USER nonroot:nonroot
ENTRYPOINT ["/app/afi-service"]
