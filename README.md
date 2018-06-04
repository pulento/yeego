# Yeego

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
docker build -t yeego .
```

Then run it:

```
docker run -it --rm --name yeego-app --net=host yeego
```

Note the `--net=host` since Yeego needs multicast to discover lights, and
do note that this network mode doesn't play nice on Docker for Mac since
its Docker inside a VM inside MacOS X :)

## Test

Use your prefered browser or just:

```
curl http://localhost:8000/lights
```
