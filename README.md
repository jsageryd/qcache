# Qcache

[![Build Status](https://github.com/jsageryd/qcache/workflows/ci/badge.svg)](https://github.com/jsageryd/qcache/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/jsageryd/qcache)](https://goreportcard.com/report/github.com/jsageryd/qcache)
[![Documentation](https://img.shields.io/badge/pkg.go.dev-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/jsageryd/qcache)
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/jsageryd/qcache/blob/master/LICENSE)

Queue-based thread-safe expiring in-memory cache.

## Usage example
```go
package main

import (
	"fmt"
	"time"

	"github.com/jsageryd/qcache"
)

func main() {
	cache := qcache.New(5 * time.Second)
	cache.Set("foo", "bar")
	if value, ok := cache.Get("foo"); ok {
		fmt.Println("Value:", value) // (type assert value as needed)
	} else {
		fmt.Println("Key not found")
	}
}
```

## Note
Setting a value twice does not reset its expiration timer.
