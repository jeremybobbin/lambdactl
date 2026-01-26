.POSIX:

CFLAGS=-Wall -Wextra -O2 -D_POSIX_C_SOURCE=200809L -Wpedantic -ggdb -fdiagnostics-color=always
LDFLAGS=-lssl -lcrypto

lambdactl:

run: lambdactl
	./lambdactl localhost:8080 /

clean:
	rm lambdactl
