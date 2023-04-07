FROM golang:1.20 AS base
    WORKDIR /go/src/github.com/CaioBittencourt/arvore-genealogica

FROM base as development
    RUN go install github.com/cortesi/modd/cmd/modd@latest && \
        go install github.com/go-delve/delve/cmd/dlv@latest
    RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.52.2

    CMD [ "modd", "-f", "./modd.conf" ]

FROM base AS compiler
    COPY . ./
    RUN go mod vendor
    RUN CGO_ENABLED=0 GOARCH=amd64 go build -o /bin/server .

FROM alpine:3.17.3 AS release
    COPY --from=compiler /bin/server /bin/server

    ENV TINI_VERSION v0.19.0
    ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static /bin/tini
    RUN chmod +x /bin/tini

    RUN addgroup -g 1000 -S user && \
        adduser -u 1000 -S user -G user
    USER user

    ENTRYPOINT ["/bin/tini", "--"]

    CMD /bin/server
