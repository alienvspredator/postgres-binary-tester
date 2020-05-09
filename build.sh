#!/bin/bash

print_usage() {
	printf "Usage:
	-t v1.0 — Tag (required)\n"
}

tag=''
while getopts 'ht:' flag; do
	case "${flag}" in
	h)
		print_usage
		exit 0
		;;
	t) tag="${OPTARG}" ;;
	?)
		print_usage
		exit 1
		;;
	esac
done

out="dist/${tag}"

GOARCH=amd64 GOOS=darwin go build -o "${out}/tester-${tag}-darwin-amd64"
GOARCH=amd64 GOOS=linux go build -o "${out}/tester-${tag}-linux-amd64"
GOARCH=386 GOOS=linux go build -o "${out}/tester-${tag}-linux-i386"
GOARCH=amd64 GOOS=windows go build -o "${out}/tester-${tag}-windows-amd64.exe"
GOARCH=386 GOOS=windows go build -o "${out}/tester-${tag}-windows-i386.exe"
