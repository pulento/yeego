FROM golang:1.10-alpine

# Install Git
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /go/src/yeego

RUN go get -d -v github.com/pulento/yeego
RUN go install -v github.com/pulento/yeego
# Put static content on WORKDIR
RUN cp -a /go/src/github.com/pulento/yeego/views .

CMD ["yeego"]
