# Wren Bindings for Go

[![GoDoc](https://godoc.org/github.com/crazyinfin8/wren-bindings-for-go)](https://pkg.go.dev/github.com/crazyinfin8/wren-bindings-for-go?tab=doc) [![Wren](https://img.shields.io/badge/github-wren-hsl(200%2C%2060%25%2C%2050%25))](https://github.com/wren-lang/wren)

Wren-bindings-for-Go provides bindings for go to interact with the [Wren](https://wren.io/) scripting language. 

This is very similar to the repo [WrenGo](https://github.com/crazyinfin8/WrenGo). While WrenGo attempts to protect the User from panics and segfaults by performing its own checks, this repo takes a much more minimalist approach and simply piping as much of the C API to the user. This means that this repo is less safe then WrenGo and can and will panic if used recklessly, but it will feel more like using the regular C API of Wren.

This API also uses a modified version of Wren that fixes some of the limitation that the api provided.

* Foreign methods and finalizers no longer need to be exposed to C allowing for users to bind unlimited foreign methods and finalizers.
* `*vm.EarlyExit()` allows a user to terminate the VM while it is running (for example, to escape from an infinite loop)
* `*vm.BytesAllocated()` allows a user to see how many bytes are allocated for Wren (at least from the C side, Go foreign objects are still stored in Go and are virtually invisible to Wren)