# unlike

Unlike old tweets.

With Twitter API v2. By default only items older than 30 days will be deleted. 
Depending on items count it can take much time. Twitter API allows 1000 requests per 24h.

## Installation
```sh
$ make
$ make install
```
or
```sh
$ brew tap x13a/tap
$ brew install x13a/tap/unlike
```

## Usage
```text
Usage of unlike:
  -V	print version and exit
  -d int
    	days to skip (default 30)
  -t duration
    	http timeout (default 30s)
  -u string
    	username (optional)
```

## Example

To unlike:
```sh
$ source ./config/unlike.env
$ unlike
```

## Friends
- [untweet](https://github.com/imwally/untweet)
- [heartbreak](https://github.com/victoriadrake/heartbreak)
