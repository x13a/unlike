# unlike

Unlike old tweets.

With Twitter API v2. By default only likes older than 7 days will be deleted.
Depending on likes count it can take much time. Twitter API allows 1000 requests per 24h.

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
    	days (default 7)
  -t duration
    	timeout (default 30s)
  -u string
    	username
```

## Example

To unlike:
```sh
$ unlike -u "USERNAME"
```

## Friends
- [untweet](https://github.com/imwally/untweet)
- [heartbreak](https://github.com/victoriadrake/heartbreak)