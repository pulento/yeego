# Yeego
[![Go Report Card](https://goreportcard.com/badge/github.com/pulento/yeego)](https://goreportcard.com/report/github.com/pulento/yeego)

Yeego is a light backend server to control Xiaomi Yeelight lights. It
presents a simple REST API for easy frontend integration.

# Install

## Native

```
go get -u github.com/pulento/yeego
go build
./yeego
```

## Docker

Build a Docker image with:

```
docker build -t pulento/yeego https://raw.githubusercontent.com/pulento/yeego/master/Dockerfile
```

Then run it:

```
docker run -it --rm --name yeego --net=host pulento/yeego
```

Note the `--net=host` since Yeego needs multicast to discover lights, and
do note that this network mode doesn't play nice on Docker for Mac since
its Docker inside a VM inside MacOS X :)

## Test

Point your prefered browser:

```
http://localhost:8080/
```
