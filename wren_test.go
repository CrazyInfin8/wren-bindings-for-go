package wren_test

import (
	_ "embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crazyinfin8/wren-bindings-for-go"
)

//go:embed test/main.wren
var test string

//go:embed test/infiniteLoop.wren
var infinitLoopTest string

//go:embed test/extras.wren
var modExtras string

//go:embed test/os.wren
var modOS string

var srcMap = map[string]string{
	"extras": modExtras,
	"os":     modOS,
}

var methodMap = map[methodKey]wren.ForeignMethodFn{
	{"os", "File", false, "read(_)"}: func(vm *wren.VM) {
		assert(vm, vm.SlotType(1) == wren.ValueTypeNum, "Expected Num for 'path'")
		var (
			content []byte
			err     error
		)
		count := int(vm.GetNum(1))
		file := vm.GetForeign(0).(*os.File)
		if count <= 0 {
			content, err = ioutil.ReadAll(file)
		} else {
			content = make([]byte, count)
			_, err = file.Read(content)
		}
		if err != nil {
			vm.SetString(0, err.Error())
			vm.AbortFiber(0)
			return
		}
		vm.SetBytes(0, content)
	},
	{"os", "Process", true, "sleep(_)"}: func(vm *wren.VM) {
		assert(vm, vm.SlotType(1) == wren.ValueTypeNum, "Expected Num for 'delay'")
		time.Sleep(time.Duration(vm.GetNum(1)) * time.Millisecond)
	},
}

var classMap = map[classKey]wren.ForeignClassMethods{
	{"os", "File"}: {
		Allocate: func(vm *wren.VM) {
			assert(vm, vm.SlotType(1) == wren.ValueTypeString, "Expected String for 'path'")
			assert(vm, vm.SlotType(2) == wren.ValueTypeNum, "Expected Num for 'flags'")
			assert(vm, vm.SlotType(3) == wren.ValueTypeNum, "Expected Num for 'permissions'")
			file, err := os.OpenFile(vm.GetString(1), int(vm.GetNum(2)), os.FileMode(vm.GetNum(3)))
			if err != nil {
				vm.SetString(0, err.Error())
				vm.AbortFiber(0)
				return
			}
			vm.SetForeign(0, 0, file)
		},
		Finalize: func(userdata interface{}) {
			if userdata.(*os.File).Close() != nil {
				println("Error closing file")
				return
			}
			println("File closed successfully")
		},
	},
}

func TestWren(t *testing.T) {
	vm := wren.Config{
		ResolveModuleFn:     resolveModuleFn,
		LoadModuleFn:        loadModuleFn,
		BindForeignMethodFn: bindForeignMethodFn,
		BindForeignClassFn:  bindForeignClassFn,
		WriteFn:             writeFn,
		ErrorFn:             errorFn,
	}.NewVM()
	defer vm.Free()

	t.Logf("Allocated at the start: %v bytes", vm.BytesAllocated())

	vm.UserData = t
	result := vm.Interpret("main.wren", test)
	if result != wren.ResultSuccess {
		t.Error("Error when running wren")
	}
	vm.EnsureSlots(2)
	vm.GetVariable("main.wren", "fn", 0)
	handle := vm.MakeCallHandle("call(_)")
	{
		defer vm.ReleaseHandle(handle)
		vm.SetString(1, "Hello from Go")
		result = vm.Call(handle)

		if result != wren.ResultSuccess {
			t.Error("Enexpected error calling handle")
		}
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		vm.EarlyExit()
		t.Log("Terminating VM prematurely")
	}()
	result = vm.Interpret("main.wren", infinitLoopTest)
	if result != wren.ResultRuntimeError {
		t.Error("Expected VM to have runtime error!")
	}
	t.Log("Wren successfully recovered from infinite loop!")

	result = vm.Interpret("other.wren", "System.print(\"does this vm still work?\")")

	switch result {
	case wren.ResultCompileError:
		t.Log("Seems like it fails with a compile error")
	case wren.ResultRuntimeError:
		t.Log("Seems like it fails with a runtime error")
	case wren.ResultSuccess:
		t.Log("Yes it does!")
	}

	t.Log("running vm.EarlyExit() when VM is off...")
	vm.EarlyExit()
	result = vm.Interpret("other-other.wren", "System.print(\"does this vm still work?\")")
	switch result {
	case wren.ResultCompileError, wren.ResultRuntimeError:
		t.Log("...exits the VM before it can do anything")
	case wren.ResultSuccess:
		t.Log("...runs the VM successfully!")
	}

	t.Logf("Allocated at the end: %v bytes", vm.BytesAllocated())
}

func resolveModuleFn(vm *wren.VM, importer, name string) (string, bool) {
	prefix := ":priv:"
	if len(name) > len(prefix) && name[:len(prefix)] == prefix {
		return name, name[len(prefix):] == importer
	}
	if _, ok := srcMap[name]; ok {
		return name, ok
	}
	if newName, err := filepath.Rel(importer, name); err != nil {
		return newName, true
	}
	return name, false
}

func loadModuleFn(vm *wren.VM, name string) (string, bool) {
	if src, ok := srcMap[name]; ok {
		return src, ok
	}
	src, err := ioutil.ReadFile(name)
	return string(src), err != nil
}

type methodKey struct {
	module, className string
	isStatic          bool
	signature         string
}

func bindForeignMethodFn(vm *wren.VM, module, className string, isStatic bool, signature string) wren.ForeignMethodFn {
	return methodMap[methodKey{
		module,
		className,
		isStatic,
		signature,
	}]
}

type classKey struct {
	module, className string
}

func bindForeignClassFn(vm *wren.VM, module, className string) wren.ForeignClassMethods {
	return classMap[classKey{module, className}]
}

func writeFn(vm *wren.VM, text string) {
	if t, ok := vm.UserData.(*testing.T); ok {
		t.Log(text)
	}
}

func errorFn(vm *wren.VM, errorType wren.ErrorType, module string, line int, msg string) {
	if t, ok := vm.UserData.(*testing.T); ok {
		switch errorType {
		case wren.ErrorCompile:
			t.Logf("[%s line %d] [Error] %s\n", module, line, msg)
		case wren.ErrorRuntime:
			t.Logf("[Runtime Error] %s\n", msg)
		case wren.ErrorStackTrace:
			t.Logf("[%s line %d] in %s\n", module, line, msg)
		}
	}
}

func assert(vm *wren.VM, condition bool, message string) {
	if !condition {
		vm.EnsureSlots(1)
		vm.SetString(0, message)
		vm.AbortFiber(0)
	}
}
