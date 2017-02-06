#This is how we want to name the binary output
BINARY=gotli

# These are the values we want to pass for Version and BuildTime
VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

Build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gotli main.go
