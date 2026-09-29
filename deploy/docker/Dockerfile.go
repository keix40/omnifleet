FROM golang:1.22-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.work buf.yaml buf.gen.yaml ./
COPY proto ./proto
COPY gen ./gen
COPY pkg ./pkg
COPY services ./services
ARG SERVICE
WORKDIR /src/services/${SERVICE}
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
