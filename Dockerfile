FROM --platform=$BUILDPLATFORM golang:1.21.3-alpine as builder
ARG TARGETOS TARGETARCH

WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build .

FROM scratch as scratch
COPY --from=builder /go/src/app/dnsupdater /bin/dnsupdater
ENTRYPOINT ["/bin/dnsupdater"]

FROM alpine:3.18 as alpine
COPY --from=builder /go/src/app/dnsupdater /bin/dnsupdater
ENTRYPOINT ["/bin/dnsupdater"]
