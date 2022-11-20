FROM golang as test

RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@latest

WORKDIR /app

ADD . .

RUN ginkgo --junit-report=report.xml ./...

FROM scratch as artifacts
COPY --from=test /app/report.xml /

FROM golang as builder

WORKDIR /accumulation-zone

ADD . .

RUN go build -v

FROM golang

WORKDIR /src
COPY --from=builder /accumulation-zone/accumulation-zone /src/

ENV PORT 80

ENTRYPOINT ["/src/accumulation-zone"]

CMD ["--help"]