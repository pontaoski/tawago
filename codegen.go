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
type namedThingImpl struct{}

func (n namedThingImpl) isNamedThing() {}

type LLVMValue struct {
	namedThingImpl
	value.Value
}
type LLVMType struct {
	namedThingImpl
	types.Type
}

type ctx struct {
	names map[Identifier]namedThing
}

func codegenExpression(c *ctx, e Expression, b *ir.Block) value.Value {
	switch expr := e.(type) {
	case Lit:
		switch lit := expr.Literal.(type) {
		case Integer:
			return constant.NewInt(types.I64, int64(lit))
		default:
			panic("unimplemented")
		}
	case Var:
		return c.names[Identifier(expr)].(LLVMValue)
	case Call:
		var args []value.Value
		for _, arg := range expr.Arguments {
			args = append(args, codegenExpression(c, arg, b))
		}
		return b.NewCall(c.names[expr.Function].(LLVMValue), args...)
	case Block:
		var last value.Value

		for _, statement := range expr {
			last = codegenExpression(c, statement, b)
		}

		b.NewRet(last)

		return last
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
	default:
		panic("unhandled")
	}

}

func codegenType(c *ctx, t Type) types.Type {
	switch kind := t.(type) {
	case Ident:
		return c.names[Identifier(kind)].(types.Type)
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

		var params []*ir.Param
		for _, param := range tl.Arguments {
			params = append(params, ir.NewParam(string(param.Name), codegenType(c, param.Kind)))
		}

		fn := m.NewFunc(string(tl.Name), ret, params...)
		bloc := fn.NewBlock("entry")

		c.names[tl.Name] = LLVMValue{Value: fn}
		retValue := codegenExpression(c, tl.Expr, bloc)
		fn.Blocks[len(fn.Blocks)-1].NewRet(retValue)
	case TypeDeclaration:
		c.names[tl.Name] = LLVMType{Type: codegenType(c, tl.Kind)}
	case Import:
		// not dealing with this
	default:
		panic("unhandled")
	}
}

func codegen(tls []TopLevel) {
	c := &ctx{
		names: map[Identifier]namedThing{
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
		},
	}

	modu := ir.NewModule()
	for _, tl := range tls {
		codegenToplevel(c, tl, modu)
	}

	fmt.Println(modu)
}
