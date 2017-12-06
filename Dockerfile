FROM golang:1.9-alpine AS BUILD

WORKDIR /go/src/github.com/sunshinekitty/vaultingkube

COPY . /go/src/github.com/sunshinekitty/vaultingkube

RUN apk update && \
    apk add curl ca-certificates git && \
    update-ca-certificates && \
    go get github.com/Masterminds/glide && \
    glide i && \
    go build -ldflags="-s -w" .

FROM alpine:latest

COPY --from=BUILD /go/src/github.com/sunshinekitty/vaultingkube/vaultingkube /usr/bin

CMD ["vaultingkube"]
