package main

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type namedThing interface{ isNamedThing() }
type NamedThingImpl struct{}

func (n NamedThingImpl) isNamedThing() {}

type LLVMMutableValue struct {
	NamedThingImpl
	value.Value
}
type LLVMValue struct {
	NamedThingImpl
	value.Value
}
type LLVMType struct {
	NamedThingImpl
	types.Type
	fields map[string]int
}

type ctx struct {
	names                  []map[Identifier]namedThing
	entry                  value.Value
	forwardDeclarationPass bool
}

func (c *ctx) pushScope() {
	c.names = append(c.names, make(map[Identifier]namedThing))
}

func (c *ctx) popScope() {
	c.names = c.names[:len(c.names)-1]
}

func (c *ctx) lookup(id Identifier) namedThing {
	for i := len(c.names) - 1; i >= 0; i-- {
		val, ok := c.names[i][id]
		if ok {
			return val
		}
	}

	panic("could not lookup " + id)
}

func (c *ctx) lookupField(t types.Type, f string) int {
	for i := len(c.names) - 1; i >= 0; i-- {
		for _, kind := range c.names[i] {
			if val, ok := kind.(LLVMType); ok {
				if val.Equal(t) {
					return val.fields[f]
				}
			}
		}
	}

	panic("could not find the given type in the current context when accessing a field")
}

func (c *ctx) assign(id Identifier, v namedThing) {
	for i := len(c.names) - 1; i >= 0; i-- {
		_, ok := c.names[i][id]
		if ok {
			c.names[i][id] = v
			return
		}
	}

	panic("could not find " + id)
}

func (c *ctx) top() map[Identifier]namedThing {
	return c.names[len(c.names)-1]
}

func codegenExpression(c *ctx, e Expression, b *ir.Block) value.Value {
	switch expr := e.(type) {
	case Lit:
		switch lit := expr.Literal.(type) {
		case Integer:
			return constant.NewInt(types.I64, int64(lit))
		case StructLiteral:
			t := c.lookup(lit.Name).(LLVMType)
			st := t.Type.(*types.StructType)

			val := b.NewAlloca(t.Type.(*types.StructType))
			for name, field := range lit.Fields {
				ptr := b.NewGetElementPtr(st, val, constant.NewInt(types.I32, int64(0)), constant.NewInt(types.I32, int64(t.fields[name])))
				expr := codegenExpression(c, field, b)
				b.NewStore(expr, ptr)
			}

			return val
		default:
			panic("unimplemented")
		}
	case Var:
		switch v := c.lookup(Identifier(expr)).(type) {
		case LLVMValue:
			return v.Value
		case LLVMMutableValue:
			return b.NewLoad(v.Value.Type().(*types.PointerType).ElemType, v.Value)
		default:
			panic("unhandled")
		}
	case Call:
		var args []value.Value
		for _, arg := range expr.Arguments {
			args = append(args, codegenExpression(c, arg, b))
		}
		return b.NewCall(c.lookup(expr.Function).(LLVMValue).Value, args...)
	case Block:
		var last value.Value

		c.pushScope()
		for _, statement := range expr {
			last = codegenExpression(c, statement, b)
		}
		c.popScope()

		return last
	case Declaration:
		val := codegenExpression(c, expr.Value, b)

		c.top()[expr.To] = LLVMValue{Value: val}

		return val
	case MutDeclaration:
		val := codegenExpression(c, expr.Value, b)

		alloca := b.NewAlloca(val.Type())
		b.NewStore(val, alloca)

		c.top()[expr.To] = LLVMMutableValue{Value: alloca}

		return val
	case Assignment:
		val := codegenExpression(c, expr.Value, b)
		b.NewStore(val, c.lookup(expr.To).(LLVMMutableValue).Value)

		return val
	case If:
		condVal := codegenExpression(c, expr.Condition, b)

		fn := b.Parent
		thenBloc := fn.NewBlock("then")
		thenValue := codegenExpression(c, expr.Then, thenBloc)

		elseBloc := fn.NewBlock("else")
		elseValue := codegenExpression(c, expr.Else, elseBloc)

		mergeBloc := fn.NewBlock("ifcont")
		phi := mergeBloc.NewPhi(ir.NewIncoming(thenValue, thenBloc), ir.NewIncoming(elseValue, elseBloc))

		// time to add the conditional now that we built the blocks
		condCmp := b.NewICmp(enum.IPredNE, condVal, constant.False)
		b.NewCondBr(condCmp, thenBloc, elseBloc)

		// now we chain the branches to the merge block
		thenBloc.NewBr(mergeBloc)
		elseBloc.NewBr(mergeBloc)

		return phi
	case Field:
		of := codegenExpression(c, expr.Of, b)
		ptr, ok := of.Type().(*types.PointerType)
		strType, strOk := ptr.ElemType.(*types.StructType)

		if !ok || !strOk {
			panic("tried to get a field of a non-struct")
		}

		field := c.lookupField(strType, string(expr.Name))

		fmt.Printf("%#v\n", ptr)
		return b.NewGetElementPtr(strType, of, constant.NewInt(types.I32, int64(0)), constant.NewInt(types.I32, int64(field)))
	default:
		panic("unhandled")
	}

}

func codegenType(c *ctx, t Type) types.Type {
	switch kind := t.(type) {
	case Ident:
		return c.lookup(Identifier(kind)).(LLVMType).Type
	case FunctionPointer:
		var ret types.Type = types.Void
		if kind.Returns != nil {
			ret = codegenType(c, *kind.Returns)
		}

		var args []types.Type
		for _, kind := range kind.Arguments {
			args = append(args, codegenType(c, kind))
		}

		return types.NewFunc(ret, args...)
	case Struct:
		var args []types.Type
		for _, kind := range kind {
			args = append(args, codegenType(c, kind.Kind))
		}

		return types.NewStruct(args...)
	default:
		panic("unhandled")
	}

}

func codegenToplevel(c *ctx, t TopLevel, m *ir.Module) {
	switch tl := t.(type) {
	case Func:
		var ret types.Type = types.Void
		if tl.Returns != nil {
			ret = codegenType(c, *tl.Returns)
		}

		if c.forwardDeclarationPass {
			var params []*ir.Param
			for _, param := range tl.Arguments {
				params = append(params, ir.NewParam(string(param.Name), codegenType(c, param.Kind)))
			}

			fn := m.NewFunc(string(tl.Name), ret, params...)
			c.top()[tl.Name] = LLVMValue{Value: fn}
			return
		}

		fn := c.lookup(tl.Name).(LLVMValue).Value.(*ir.Func)
		bloc := fn.NewBlock("entry")

		if tl.Name == "main" {
			c.entry = fn
		}

		c.pushScope()
		for i, arg := range tl.Arguments {
			c.top()[arg.Name] = LLVMValue{Value: fn.Params[i]}
		}
		retValue := codegenExpression(c, tl.Expr, bloc)
		c.popScope()

		if types.IsVoid(ret) {
			fn.Blocks[len(fn.Blocks)-1].NewRet(nil)
		} else {
			fn.Blocks[len(fn.Blocks)-1].NewRet(retValue)
		}
	case TypeDeclaration:
		c.top()[tl.Name] = LLVMType{Type: codegenType(c, tl.Kind)}
		if v, ok := tl.Kind.(Struct); ok {
			t := c.top()[tl.Name].(LLVMType)
			t.Type.SetName(string(tl.Name))
			t.fields = make(map[string]int)
			for idx, field := range v {
				t.fields[field.Name] = idx
			}
			c.top()[tl.Name] = t
		}
	case Import:
		// not dealing with this
	default:
		panic("unhandled")
	}
}

func codegen(tls []TopLevel) *ir.Module {
	c := &ctx{
		names: []map[Identifier]namedThing{
			{
				"int8":   LLVMType{Type: types.I8},
				"int16":  LLVMType{Type: types.I16},
				"int32":  LLVMType{Type: types.I32},
				"int64":  LLVMType{Type: types.I64},
				"int128": LLVMType{Type: types.I128},

				"float16":  LLVMType{Type: types.Half},
				"float32":  LLVMType{Type: types.Float},
				"float64":  LLVMType{Type: types.Double},
				"float128": LLVMType{Type: types.FP128},

				"bool":  LLVMType{Type: types.I1},
				"niets": LLVMType{Type: types.Void},

				"true":  LLVMValue{Value: constant.False},
				"false": LLVMValue{Value: constant.True},

				"nil": LLVMValue{Value: nil},
			},
		},
	}

	modu := ir.NewModule()
	c.forwardDeclarationPass = true
	for _, tl := range tls {
		codegenToplevel(c, tl, modu)
	}
	c.forwardDeclarationPass = false
	for _, tl := range tls {
		if _, ok := tl.(Func); ok {
			codegenToplevel(c, tl, modu)
		}
	}

	if c.entry != nil {
		opening := modu.NewFunc("_tawa_main", types.Void)
		bloc := opening.NewBlock("_entry")

		bloc.NewCall(c.entry)
		bloc.NewCall(ir.NewInlineAsm(types.NewPointer(types.NewFunc(types.Void)), `movl $$0x1, %eax; movl $$0x1, %ebx; int $$0x80`, ``))
		bloc.NewRet(nil)
	}

	return modu
}
