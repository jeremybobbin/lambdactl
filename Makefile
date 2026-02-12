.POSIX:

CFLAGS=-Wall -Wextra -O2 -D_POSIX_C_SOURCE=200809L -Wpedantic -ggdb -fdiagnostics-color=always
LDFLAGS=-lssl -lcrypto

build: lambdactl menu

lambdactl:
menu:

run: lambdactl menu
	./lambdactl localhost:8080 /

clean:
	rm lambdactl
