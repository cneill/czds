# czds

Download zone files from ICANN's CZDS portal. Based on ICANN's [czds-api-client-python](https://github.com/icann/czds-api-client-python).

## Setup 

```
git clone git@github.com:cneill/czds.git
mv config.dist.json config.json
vi config.json
go build
```

## Usage

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
