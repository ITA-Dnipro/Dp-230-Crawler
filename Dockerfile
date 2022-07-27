FROM golang:1.18 as builder
WORKDIR /go/src
COPY . .
RUN make test-coverage
RUN make build

FROM alpine
COPY --from=builder /go/src/bin/crawler /usr/bin
ENTRYPOINT [ "crawler" ]