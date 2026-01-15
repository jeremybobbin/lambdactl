.POSIX:

CFLAGS=-Wall -Wextra -O2 -D_POSIX_C_SOURCE=200112L
LDFLAGS=-lssl -lcrypto

lambdactl:

run: lambdactl
	./lambdactl localhost:8080 /
