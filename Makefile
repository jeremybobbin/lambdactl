.POSIX:

build:
	go build

run: build
	./lambdactl
