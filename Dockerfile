FROM golang:1.10

WORKDIR /go/src/yeego
COPY . .

RUN go get -d -v github.com/pulento/yeego
RUN go install -v github.com/pulento/yeego

CMD ["yeego"]
