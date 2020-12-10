package main

import (
	"crypto/md5"

	hash_stuff "github.com/davidwallacejackson/hash-stuff"
)

var dir = ""
var include = []string{}
var exclude = []string{}

func main() {
	list, err := hash_stuff.ListFiles(dir, include, exclude)
	if err != nil {
		panic(err)
	}

	x := md5.New()
	println(x)

	for _, path := range list {
		println(path)
	}
}
