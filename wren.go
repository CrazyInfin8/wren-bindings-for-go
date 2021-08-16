package wren

/*
#cgo CFLAGS:
#cgo LDFLAGS: -lm
#include "wren.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

//go:generate go run getWren.go

var vmMap = make(map[*C.WrenVM]*VM)
var vmMux sync.RWMutex

const (
	VersionMajor  int    = C.WREN_VERSION_MAJOR
	VersionMinor  int    = C.WREN_VERSION_MINOR
	VersionPatch  int    = C.WREN_VERSION_PATCH
	VersionNumber int    = C.WREN_VERSION_NUMBER
	VersionString string = C.WREN_VERSION_STRING
)

func versionTuple() [3]int {
	return [3]int{
		C.WREN_VERSION_MAJOR,
		C.WREN_VERSION_MINOR,
		C.WREN_VERSION_PATCH,
	}
}

type VM struct {
	vm          *C.WrenVM
	config      Config
	foreignMap  map[unsafe.Pointer]interface{}
	functionMap map[unsafe.Pointer]ForeignMethodFn
	finalizeMap map[unsafe.Pointer]FinalizerFn
	UserData    interface{}
}

type Handle C.WrenHandle

type ForeignMethodFn func(vm *VM)

type FinalizerFn func(userdata interface{})

type ResolveModuleFn func(vm *VM, importer, name string) (newName string, ok bool)

type LoadModuleFn func(vm *VM, name string) (src string, ok bool)

type BindForeignMethodFn func(vm *VM, module, className string, isStatic bool, signature string) ForeignMethodFn

type WriteFn func(vm *VM, text string)

type ErrorType C.WrenErrorType

const (
	ErrorCompile    = C.WREN_ERROR_COMPILE
	ErrorRuntime    = C.WREN_ERROR_RUNTIME
	ErrorStackTrace = C.WREN_ERROR_STACK_TRACE
)

type ErrorFn func(vm *VM, errorType ErrorType, module string, line int, message string)

type ForeignClassMethods struct {
	Allocate ForeignMethodFn
	Finalize FinalizerFn
}

type BindForeignClassFn func(vm *VM, module, className string) ForeignClassMethods

type Config struct {
	ResolveModuleFn     ResolveModuleFn
	LoadModuleFn        LoadModuleFn
	BindForeignMethodFn BindForeignMethodFn
	BindForeignClassFn  BindForeignClassFn
	WriteFn             WriteFn
	ErrorFn             ErrorFn
	InitialHeapSize     int64
	MinHeapSize         int64
	HeapGrowthPercent   int
}

type InterpretResult C.WrenInterpretResult

const (
	ResultSuccess      InterpretResult = C.WREN_RESULT_SUCCESS
	ResultCompileError InterpretResult = C.WREN_RESULT_COMPILE_ERROR
	ResultRuntimeError InterpretResult = C.WREN_RESULT_RUNTIME_ERROR
)

type ValueType C.WrenType

const (
	ValueTypeBool    ValueType = C.WREN_TYPE_BOOL
	ValueTypeNum     ValueType = C.WREN_TYPE_NUM
	ValueTypeForeign ValueType = C.WREN_TYPE_FOREIGN
	ValueTypeList    ValueType = C.WREN_TYPE_MAP
	ValueTypeMap     ValueType = C.WREN_TYPE_MAP
	ValueTypeNull    ValueType = C.WREN_TYPE_NULL
	ValueTypeString  ValueType = C.WREN_TYPE_STRING

	ValueTypeUnknown ValueType = C.WREN_TYPE_UNKNOWN
)

func NewVM() *VM {
	return &VM{vm: C.wrenNewVM(nil)}
}

func (config Config) NewVM() *VM {
	var cfg C.WrenConfiguration
	setConfiguration(&cfg, &config)

	wrenVM := C.wrenNewVM((*C.WrenConfiguration)(&cfg))
	vm := &VM{
		vm:          wrenVM,
		config:      config,
		foreignMap:  make(map[unsafe.Pointer]interface{}),
		functionMap: make(map[unsafe.Pointer]ForeignMethodFn),
		finalizeMap: make(map[unsafe.Pointer]FinalizerFn),
	}

	vmMux.Lock()
	vmMap[wrenVM] = vm
	vmMux.Unlock()

	return vm
}

func (vm *VM) Free() {
	C.wrenFreeVM(vm.vm)
	if vm.functionMap != nil {
		for key := range vm.functionMap {
			delete(vm.functionMap, key)
			C.free(key)
		}
	}
	if vm.finalizeMap != nil {
		for key := range vm.finalizeMap {
			delete(vm.finalizeMap, key)
			C.free(key)
		}
	}
	vmMux.Lock()
	delete(vmMap, vm.vm)
	vmMux.Unlock()
}

func (vm *VM) EarlyExit() {
	C.wrenEarlyExit(vm.vm)
}

func (vm *VM)BytesAllocated() uint {
	return uint(C.wrenGetAllocated(vm.vm))
}

func (vm *VM) CollectGarbage() {
	C.wrenCollectGarbage(vm.vm)
}

func (vm *VM) Interpret(module, source string) InterpretResult {
	cMod, cSrc := C.CString(module), C.CString(source)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cSrc))
	}()
	return InterpretResult(C.wrenInterpret(vm.vm, cMod, cSrc))
}

func (vm *VM) MakeCallHandle(signature string) *Handle {
	cSig := C.CString(signature)
	defer C.free(unsafe.Pointer(cSig))
	return (*Handle)(C.wrenMakeCallHandle(vm.vm, cSig))
}

func (vm *VM) Call(handle *Handle) InterpretResult {
	return InterpretResult(C.wrenCall(vm.vm, (*C.WrenHandle)(handle)))
}

func (vm *VM) ReleaseHandle(handle *Handle) {
	C.wrenReleaseHandle(vm.vm, (*C.WrenHandle)(handle))
}

func (vm *VM) SlotCount() int {
	return int(C.wrenGetSlotCount(vm.vm))
}

func (vm *VM) EnsureSlots(numSlots int) {
	C.wrenEnsureSlots(vm.vm, C.int(numSlots))
}

func (vm *VM) SlotType(slot int) ValueType {
	return ValueType(C.wrenGetSlotType(vm.vm, C.int(slot)))
}

func (vm *VM) GetBool(slot int) bool {
	return bool(C.wrenGetSlotBool(vm.vm, C.int(slot)))
}

func (vm *VM) GetBytes(slot int) []byte {
	var length C.int
	str := C.wrenGetSlotBytes(vm.vm, C.int(slot), &length)
	return C.GoBytes(unsafe.Pointer(str), length)
}

func (vm *VM) GetNum(slot int) float64 {
	return float64(C.wrenGetSlotDouble(vm.vm, C.int(slot)))
}

func (vm *VM) GetForeign(slot int) interface{} {
	ptr := C.wrenGetSlotForeign(vm.vm, C.int(slot))
	return vm.foreignMap[ptr]
}

func (vm *VM) GetString(slot int) string {
	var length C.int
	str := C.wrenGetSlotBytes(vm.vm, C.int(slot), &length)
	return C.GoStringN(str, length)
}

func (vm *VM) GetHandle(slot int) *Handle {
	return (*Handle)(C.wrenGetSlotHandle(vm.vm, C.int(slot)))
}

func (vm *VM) SetBool(slot int, value bool) {
	C.wrenSetSlotBool(vm.vm, C.int(slot), C.bool(value))
}

func (vm *VM) SetBytes(slot int, bytes []byte) {
	str := C.CBytes(bytes)
	defer C.free(str)
	C.wrenSetSlotBytes(vm.vm, C.int(slot), (*C.char)(str), C.size_t(len(bytes)))
}

func (vm *VM) SetNum(slot int, value float64) {
	C.wrenSetSlotDouble(vm.vm, C.int(slot), C.double(value))
}

func (vm *VM) SetForeign(slot, classSlot int, value interface{}) {
	ptr := C.wrenSetSlotNewForeign(vm.vm, C.int(slot), C.int(classSlot), 1)
	vm.foreignMap[ptr] = value
}

func (vm *VM) SetNewList(slot int) {
	C.wrenSetSlotNewList(vm.vm, C.int(slot))
}

func (vm *VM) SetNewMap(slot int) {
	C.wrenSetSlotNewMap(vm.vm, C.int(slot))
}

func (vm *VM) SetNull(slot int) {
	C.wrenSetSlotNull(vm.vm, C.int(slot))
}

func (vm *VM) SetString(slot int, text string) {
	str := C.CString(text)
	defer C.free(unsafe.Pointer(str))
	C.wrenSetSlotBytes(vm.vm, C.int(slot), (*C.char)(str), C.size_t(len(text)))
}

func (vm *VM) SetHandle(slot int, handle *Handle) {
	C.wrenSetSlotHandle(vm.vm, C.int(slot), (*C.WrenHandle)(handle))
}

func (vm *VM) ListCount(slot int) int {
	return int(C.wrenGetListCount(vm.vm, C.int(slot)))
}

func (vm *VM) GetListElement(listSlot, index, elementSlot int) {
	C.wrenGetListElement(vm.vm, C.int(listSlot), C.int(index), C.int(elementSlot))
}

func (vm *VM) SetListElement(listSlot, index, elementSlot int) {
	C.wrenSetListElement(vm.vm, C.int(listSlot), C.int(index), C.int(elementSlot))
}

func (vm *VM) InsertInList(listSlot, index, elementSlot int) {
	C.wrenInsertInList(vm.vm, C.int(listSlot), C.int(index), C.int(elementSlot))
}

func (vm *VM) MapCount(slot int) int {
	return int(C.wrenGetMapCount(vm.vm, C.int(slot)))
}

func (vm *VM) MapHasKey(mapSlot, keySlot int) bool {
	return bool(C.wrenGetMapContainsKey(vm.vm, C.int(mapSlot), C.int(keySlot)))
}

func (vm *VM) GetMapValue(mapSlot, keySlot, valueSlot int) {
	C.wrenGetMapValue(vm.vm, C.int(mapSlot), C.int(keySlot), C.int(valueSlot))
}

func (vm *VM) SetMapValue(mapSlot, keySlot, valueSlot int) {
	C.wrenSetMapValue(vm.vm, C.int(mapSlot), C.int(keySlot), C.int(valueSlot))
}

func (vm *VM) RemoveMapValue(mapSlot, keySlot, removedValueSlot int) {
	C.wrenRemoveMapValue(vm.vm, C.int(mapSlot), C.int(keySlot), C.int(removedValueSlot))
}

func (vm *VM) GetVariable(module, name string, slot int) {
	cMod, cName := C.CString(module), C.CString(name)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cName))
	}()
	C.wrenGetVariable(vm.vm, cMod, cName, C.int(slot))
}

func (vm *VM) HasVariable(module, name string) {
	cMod, cName := C.CString(module), C.CString(name)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cName))
	}()
	C.wrenHasVariable(vm.vm, cMod, cName)
}

func (vm *VM) GetIfHasVariable(module, name string, slot int) bool {
	cMod, cName := C.CString(module), C.CString(name)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cName))
	}()
	if C.wrenHasVariable(vm.vm, cMod, cName) {
		C.wrenGetVariable(vm.vm, cMod, cName, C.int(slot))
		return true
	}
	return false
}

func (vm *VM) HasModule(module string) bool {
	cMod := C.CString(module)
	defer C.free(unsafe.Pointer(cMod))
	return bool(C.wrenHasModule(vm.vm, cMod))
}

func (vm *VM) AbortFiber(slot int) {
	C.wrenAbortFiber(vm.vm, C.int(slot))
}
