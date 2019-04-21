# czds

Download zone files from ICANN's CZDS portal. Based on ICANN's [czds-api-client-python](https://github.com/icann/czds-api-client-python).

## Setup 

```
git clone git@github.com:cneill/czds.git
mv config.dist.json config.json
vi config.json
go build
```

Alternatively, create a `config.json` file on your own based on [config.dist.json](./config.dist.json) and install with:

```
go get -u github.com/cneill/czds
```

## Usage

```
Usage: czds [GLOBAL OPTIONS] <COMMAND> [COMMAND OPTIONS]
Commands:
* download
* list
* parse

Global options:
  -config string
        config file to load (default "config.json")
  -verbose
        verbose output
```

### List available zone files

```
./czds list
```

### Download zone files

```
./czds download
```

### Parse zone files

```
./czds parse
```

## License

Copyright (c) 2019 Charles Neill. All rights reserved. czds is distributed under an open-source BSD licence.
