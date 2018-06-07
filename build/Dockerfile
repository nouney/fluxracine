FROM golang:1.9.4
WORKDIR /go/src/github.com/nouney/fluxracine
ADD . .
WORKDIR /go/src/github.com/nouney/fluxracine/cmd/webchat
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server *.go

FROM drone/ca-certs
ARG APP_PATH
WORKDIR /
COPY --from=0 /go/src/github.com/nouney/fluxracine/cmd/webchat/server .
ENTRYPOINT ["/server"]
