FROM golang as test

RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@v2.6.1 \
    && go install github.com/golang/mock/mockgen@v1.6.0 \
    && go install github.com/t-yuki/gocover-cobertura@latest

WORKDIR /app

ADD . .

RUN go generate ./... && ginkgo --cover --junit-report=report.xml ./...
RUN gocover-cobertura < coverprofile.out > coverprofile.xml

FROM scratch as artifacts
COPY --from=test /app/report.xml /
COPY --from=test /app/coverprofile.xml /
COPY --from=test /app/coverprofile.out /

FROM golang as builder

WORKDIR /accumulation-zone

ADD . .

RUN go build -v

FROM golang

WORKDIR /src
COPY --from=builder /accumulation-zone/accumulation-zone /src/

VOLUME /data
ENV PATH_TO_BACKUP /data

ENTRYPOINT ["/src/accumulation-zone"]

CMD ["--help"]