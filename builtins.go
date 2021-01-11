package main

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func getStructElm(b *ir.Block, t types.Type, v value.Value, idx int64) value.Value {
	return b.NewGetElementPtr(t, v, constant.NewInt(types.I32, int64(0)), constant.NewInt(types.I32, int64(idx)))
}

func addBuiltins(m *ir.Module) (ret map[string]value.Value) {
	ret = make(map[string]value.Value)

	funcs := []func(*ir.Module) (string, value.Value){
		addPrint,
	}
	for _, fn := range funcs {
		k, v := fn(m)
		ret[k] = v
	}

	return
}

func addPrint(m *ir.Module) (string, value.Value) {
	fn := m.NewFunc("print", types.Void, ir.NewParam("input", types.NewPointer(String.Type)))
	entry := fn.NewBlock("entry")

	len := getStructElm(entry, String.Type, fn.Params[0], 0)
	loadedLen := entry.NewLoad(Int64.Type, len)
	data := getStructElm(entry, String.Type, fn.Params[0], 1)
	loadedData := entry.NewLoad(types.NewPointer(Byte), data)

	asm := ir.NewInlineAsm(
		types.NewPointer(types.NewFunc(types.Void, Int64.Type, types.NewPointer(Byte))),
		`movq $0, %rsi; movq $1, %rdx; movq $$0x1, %rax; movq $$0x1, %rdi; syscall`,
		`r,r`,
	)
	asm.SideEffect = true

	entry.NewCall(asm, loadedData, loadedLen)
	entry.NewRet(nil)

	return "print", fn
}
