# lambdactl
## interactive terminal menu for controlling lambda cloud instances

### dependencies

```
curl
jq
coreutils
awk
```

### compile

```
make
```

### settings

```sh
export LAMBDA_API_KEY="..."
```


### run

```
./lambdactl
```

### install

```
make clean &&
	make BIN=/usr/local/bin LIB=/usr/local/lib/lambdactl &&
	sudo make install BIN=/usr/local/bin LIB=/usr/local/lib/lambdactl
```
