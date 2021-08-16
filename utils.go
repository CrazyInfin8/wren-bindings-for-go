package wren

/*
#cgo CFLAGS:
#cgo LDFLAGS: -lm
#include "wren.h"

extern char* resolveModuleFn(WrenVM*, char*, char*);
extern WrenLoadModuleResult loadModuleFn(WrenVM*, char*);
extern void loadModuleCompleteFn(WrenVM*, char*, WrenLoadModuleResult);
extern WrenBindForeignMethodResult bindForeignMethodFn(WrenVM*, char*, char*, bool, char*);
extern WrenForeignClassMethods bindForeignClassFn(WrenVM*, char*, char*);
extern void errorFn(WrenVM*, WrenErrorType, char*, int, char*);
extern void writeFn(WrenVM*, char*);

extern void finalizeFn(WrenVM*, void*, void*);
extern void executeFn(WrenVM*, void*);
*/
import "C"

import (
	"unsafe"
)

type functionKey struct {
	module    string
	className string
	signatuer string
	isStatic  bool
}

type classKey struct {
	module    string
	className string
}

func setConfiguration(cfg *C.WrenConfiguration, config *Config) {
	C.wrenInitConfiguration(cfg)
	// resolveModule
	if config.ResolveModuleFn != nil {
		cfg.resolveModuleFn = C.WrenResolveModuleFn(C.resolveModuleFn)
	}
	if config.LoadModuleFn != nil {
		cfg.loadModuleFn = C.WrenLoadModuleFn(C.loadModuleFn)
	}
	if config.BindForeignMethodFn != nil {
		cfg.bindForeignMethodFn = C.WrenBindForeignMethodFn(C.bindForeignMethodFn)
	}
	if config.BindForeignClassFn != nil {
		cfg.bindForeignClassFn = C.WrenBindForeignClassFn(C.bindForeignClassFn)
	}
	if config.WriteFn != nil {
		cfg.writeFn = C.WrenWriteFn(C.writeFn)
	}
	if config.ErrorFn != nil {
		cfg.errorFn = C.WrenErrorFn(C.errorFn)
	}
	if config.InitialHeapSize != 0 {
		cfg.initialHeapSize = C.size_t(config.InitialHeapSize)
	}
	if config.MinHeapSize != 0 {
		cfg.minHeapSize = C.size_t(config.InitialHeapSize)
	}
	if cfg.heapGrowthPercent != 0 {
		cfg.heapGrowthPercent = C.int(config.HeapGrowthPercent)
	}
}

//export resolveModuleFn
func resolveModuleFn(wrenVM *C.WrenVM, importer, name *C.char) *C.char {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	if newName, ok := vm.config.ResolveModuleFn(vm, C.GoString(importer), C.GoString(name)); ok {
		// wren frees this value
		return C.CString(newName)
	}
	return nil
}

//export loadModuleFn
func loadModuleFn(wrenVM *C.WrenVM, name *C.char) C.WrenLoadModuleResult {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	if src, ok := vm.config.LoadModuleFn(vm, C.GoString(name)); ok {
		csrc := C.CString(src)
		return C.WrenLoadModuleResult{
			source:     csrc,
			onComplete: C.WrenLoadModuleCompleteFn(C.loadModuleCompleteFn),
			userData:   nil,
		}
	}
	return C.WrenLoadModuleResult{}
}

//export loadModuleCompleteFn
func loadModuleCompleteFn(vm *C.WrenVM, name *C.char, result C.WrenLoadModuleResult) {
	C.free(unsafe.Pointer(result.source))
}

//export bindForeignMethodFn
func bindForeignMethodFn(wrenVM *C.WrenVM, module, className *C.char, isStatic C.bool, signature *C.char) C.WrenBindForeignMethodResult {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	if method := vm.config.BindForeignMethodFn(vm, C.GoString(module), C.GoString(className), bool(isStatic), C.GoString(signature)); method == nil {
		return C.WrenBindForeignMethodResult{}
	} else {
		functionKey := C.malloc(1)
		vm.functionMap[functionKey] = method

		return C.WrenBindForeignMethodResult{
			executeFn: C.WrenForeignMethodFn(C.executeFn),
			userData:  functionKey,
		}
	}
}

//export bindForeignClassFn
func bindForeignClassFn(wrenVM *C.WrenVM, module, className *C.char) C.WrenForeignClassMethods {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	if fcm := vm.config.BindForeignClassFn(vm, C.GoString(module), C.GoString(className)); fcm.Allocate == nil && fcm.Finalize == nil {
		return C.WrenForeignClassMethods{}
	} else {
		allocateKey := C.malloc(1)
		vm.functionMap[allocateKey] = fcm.Allocate

		finalizeKey := C.malloc(1)
		vm.finalizeMap[finalizeKey] = fcm.Finalize

		return C.WrenForeignClassMethods{
			allocate:         C.WrenForeignMethodFn(C.executeFn),
			finalize:         C.WrenFinalizerFn(C.finalizeFn),
			allocateUserData: allocateKey,
			finalizeUserData: finalizeKey,
		}
	}
}

//export finalizeFn
func finalizeFn(wrenVM *C.WrenVM, ptr unsafe.Pointer, userData unsafe.Pointer) {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	vm.finalizeMap[userData](vm.foreignMap[ptr])
	delete(vm.foreignMap, ptr)
}

//export executeFn
func executeFn(wrenVM *C.WrenVM, userData unsafe.Pointer) {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	vm.functionMap[userData](vm)
}

//export writeFn
func writeFn(wrenVM *C.WrenVM, text *C.char) {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	vm.config.WriteFn(vm, C.GoString(text))
}

//export errorFn
func errorFn(wrenVM *C.WrenVM, errorType C.WrenErrorType, module *C.char, line C.int, message *C.char) {
	vmMux.RLock()
	vm := vmMap[wrenVM]
	vmMux.RUnlock()

	vm.config.ErrorFn(vm, ErrorType(errorType), C.GoString(module), int(line), C.GoString(message))
}
