.POSIX:

CC=/bin/cc
BIN=./
LIB=./
CFLAGS=-Wall -Wextra -O2 -Wpedantic -ggdb -fdiagnostics-color=always -DLIB=\"$(LIB)\" -DBIN=\"$(BIN)\"
LDFLAGS=

build: lambdactl menu

lambdactl:
menu:

run: lambdactl menu
	./lambdactl localhost:8080 /

clean:
	rm -f lambdactl menu

install: lambdactl menu
	mkdir -p $(LIB)
	cp lambdactl $(BIN)
	cp -a bin menu $(LIB)
