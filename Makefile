.POSIX:

CC=/bin/cc
CFLAGS=-Wall -Wextra -O2 -Wpedantic -ggdb -fdiagnostics-color=always
LDFLAGS=

build: lambdactl menu

lambdactl:
menu:

run: lambdactl menu
	./lambdactl localhost:8080 /

clean:
	rm lambdactl menu
