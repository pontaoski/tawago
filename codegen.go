package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"

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

type uerror interface {
	UError() string
}
type uerrorImpl struct {
	msg string
}

func (u uerrorImpl) UError() string {
	return u.msg
}

func NewUError(msg string, fmts ...interface{}) uerror {
	a := uerrorImpl{}
	a.msg = fmt.Sprintf(msg, fmts...)

	return a
}

type ctx struct {
	names                  []map[string]namedThing
	entry                  value.Value
	forwardDeclarationPass bool
}

func (c *ctx) pushScope() {
	c.names = append(c.names, make(map[string]namedThing))
}

func (c *ctx) popScope() {
	c.names = c.names[:len(c.names)-1]
}

func (c *ctx) lookup(id Identifier) namedThing {
	for i := len(c.names) - 1; i >= 0; i-- {
		val, ok := c.names[i][id.Name]
		if ok {
			return val
		}
	}

	panic("could not lookup " + id.Name)
}

func (c *ctx) lookupField(t types.Type, f string) (int, error) {
	for i := len(c.names) - 1; i >= 0; i-- {
		for _, kind := range c.names[i] {
			if val, ok := kind.(LLVMType); ok {
				if val.Equal(t) {
					return val.fields[f], nil
				}
			}
		}
	}

	return -1, errors.New("could not find the given type in the current context when accessing a field")
}

func (c *ctx) assign(id Identifier, v namedThing) {
	for i := len(c.names) - 1; i >= 0; i-- {
		_, ok := c.names[i][id.Name]
		if ok {
			c.names[i][id.Name] = v
			return
		}
	}

	panic("could not find " + id.Name)
}

func (c *ctx) top() map[string]namedThing {
	return c.names[len(c.names)-1]
}

func posOf(e Expression) Span {
	defer func() {
		recover()
	}()

	v := reflect.ValueOf(e)

	pos := v.FieldByName("Pos")
	if pos.IsZero() {
		return Span{}
	}

	return pos.Interface().(Span)
}

func codegenExpression(c *ctx, e Expression, b *ir.Block) value.Value {
	switch expr := e.(type) {
	case Lit:
		switch lit := expr.Literal.(type) {
		case Integer:
			return constant.NewInt(Int64.Type.(*types.IntType), int64(lit))
		case StructLiteral:
			t := c.lookup(lit.Name).(LLVMType)
			st := t.Type.(*types.StructType)

			val := b.NewAlloca(t.Type.(*types.StructType))
			for name, field := range lit.Fields {
				ptr := b.NewGetElementPtr(st, val, constant.NewInt(types.I32, int64(0)), constant.NewInt(types.I32, int64(t.fields[name])))
				expr := codegenExpression(c, field, b)

				fieldType := st.Fields[t.fields[name]]
				if !fieldType.Equal(expr.Type()) {
					panic(NewUError("%s: field '%s' has type '%s', not type '%s'", posOf(field), name, fieldType.Name(), expr.Type().Name()))
				}

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

		c.top()[expr.To.Name] = LLVMValue{Value: val}

		return val
	case MutDeclaration:
		val := codegenExpression(c, expr.Value, b)

		alloca := b.NewAlloca(val.Type())
		b.NewStore(val, alloca)

		c.top()[expr.To.Name] = LLVMMutableValue{Value: alloca}

		return val
	case Assignment:
		val := codegenExpression(c, expr.Value, b)
		to, ok := c.lookup(expr.To).(LLVMMutableValue)
		if !ok {
			panic(NewUError("%s: %s is not mutable", expr.Pos, expr.To))
		}

		valType := val.Type()
		elmType := to.Type().(*types.PointerType).ElemType
		if !val.Type().Equal(elmType) {
			if ptr, ok := valType.(*types.PointerType); ok {
				valType = ptr.ElemType
			}
			if ptr, ok := elmType.(*types.PointerType); ok {
				elmType = ptr.ElemType
			}
			panic(NewUError("%s: tried to assign something of type '%s' to type '%s'", expr.Pos, valType.Name(), elmType.Name()))
		}
		b.NewStore(val, to.Value)

		return val
	case FieldAssignment:
		val := codegenExpression(c, expr.Value, b)

		of := codegenExpression(c, expr.Struct, b)
		ptr, ok := of.Type().(*types.PointerType)
		strType, strOk := ptr.ElemType.(*types.StructType)

		if !ok || !strOk {
			panic(NewUError("%s: tried to assign to a field of a non-struct", expr.Pos))
		}

		field, err := c.lookupField(strType, string(expr.Field.Name))
		if err != nil {
			panic(NewUError("%s: struct type '%s' does not have field '%s'", expr.Pos, strType.Name(), expr.Field))
		}

		if !strType.Equal(val.Type()) {
			panic(NewUError("%s: field '%s' has type '%s', not type '%s'", expr.Pos, expr.Field, strType.Fields[field].Name(), val.Type().Name()))
		}

		eep := b.NewGetElementPtr(strType, of, constant.NewInt(types.I32, int64(0)), constant.NewInt(types.I32, int64(field)))

		b.NewStore(val, eep)
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

		field, err := c.lookupField(strType, string(expr.Field.Name))
		if err != nil {
			panic(NewUError("struct type '%s' does not have field '%s'", strType.Name(), expr.Field))
		}

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
				params = append(params, ir.NewParam(string(param.Ident.Name), codegenType(c, param.Kind)))
			}

			fn := m.NewFunc(string(tl.Ident.Name), ret, params...)
			c.top()[tl.Ident.Name] = LLVMValue{Value: fn}
			return
		}

		fn := c.lookup(tl.Ident).(LLVMValue).Value.(*ir.Func)
		bloc := fn.NewBlock("entry")

		if tl.Ident.Name == "main" {
			c.entry = fn
		}

		c.pushScope()
		for i, arg := range tl.Arguments {
			c.top()[arg.Ident.Name] = LLVMValue{Value: fn.Params[i]}
		}
		retValue := codegenExpression(c, tl.Expr, bloc)
		c.popScope()

		if types.IsVoid(ret) {
			fn.Blocks[len(fn.Blocks)-1].NewRet(nil)
		} else {
			fn.Blocks[len(fn.Blocks)-1].NewRet(retValue)
		}
	case TypeDeclaration:
		c.top()[tl.Ident.Name] = LLVMType{Type: codegenType(c, tl.Kind)}
		if v, ok := tl.Kind.(Struct); ok {
			t := c.top()[tl.Ident.Name].(LLVMType)
			t.Type.SetName(string(tl.Ident.Name))
			m.TypeDefs = append(m.TypeDefs, t.Type)
			t.fields = make(map[string]int)
			for idx, field := range v {
				t.fields[field.Name] = idx
			}
			c.top()[tl.Ident.Name] = t
		}
	case Import:
		// not dealing with this
	default:
		panic("unhandled")
	}
}

func codegen(tls []TopLevel) *ir.Module {
	defer func() {
		if v := recover(); v != nil {
			if uerror, ok := v.(uerror); ok {
				println(uerror.UError())
				os.Exit(1)
			} else {
				panic(v)
			}
		}
	}()

	c := &ctx{
		names: []map[string]namedThing{
			{
				"int8":     Int8,
				"int16":    Int16,
				"int32":    Int32,
				"int64":    Int64,
				"int128":   Int128,
				"float16":  Float16,
				"float32":  Float32,
				"float64":  Float64,
				"float128": Float128,
				"bool":     Boolean,
				"niets":    Niets,

				"true":  LLVMValue{Value: constant.NewInt(Boolean.Type.(*types.IntType), 1)},
				"false": LLVMValue{Value: constant.NewInt(Boolean.Type.(*types.IntType), 0)},
				"nil":   LLVMValue{Value: nil},
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
