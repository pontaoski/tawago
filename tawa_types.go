package main

import (
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

var (
	Int8   = LLVMType{Type: &types.IntType{BitSize: 8, TypeName: "int8"}}
	Int16  = LLVMType{Type: &types.IntType{BitSize: 16, TypeName: "int16"}}
	Int32  = LLVMType{Type: &types.IntType{BitSize: 32, TypeName: "int32"}}
	Int64  = LLVMType{Type: &types.IntType{BitSize: 64, TypeName: "int64"}}
	Int128 = LLVMType{Type: &types.IntType{BitSize: 128, TypeName: "int128"}}

	Float16  = LLVMType{Type: &types.FloatType{Kind: types.FloatKindHalf, TypeName: "float16"}}
	Float32  = LLVMType{Type: &types.FloatType{Kind: types.FloatKindFloat, TypeName: "float32"}}
	Float64  = LLVMType{Type: &types.FloatType{Kind: types.FloatKindDouble, TypeName: "float64"}}
	Float128 = LLVMType{Type: &types.FloatType{Kind: types.FloatKindFP128, TypeName: "float128"}}

	Byte = LLVMType{Type: &types.IntType{BitSize: 8, TypeName: "byte"}}

	Boolean = LLVMType{Type: &types.IntType{BitSize: 1, TypeName: "bool"}}
	Niets   = LLVMType{Type: &types.VoidType{TypeName: "niets"}}

	String = LLVMType{
		Type: func() types.Type {
			strct := types.NewStruct(
				Int64.Type,
				types.NewPointer(Byte),
			)
			strct.SetName("string_impl")

			return strct
		}(),
	}
	StringPointer = LLVMType{
		Type: types.NewPointer(String),
	}
)

var (
	True  = LLVMValue{Value: constant.NewInt(Boolean.Type.(*types.IntType), 1)}
	False = LLVMValue{Value: constant.NewInt(Boolean.Type.(*types.IntType), 0)}
	Nil   = LLVMValue{Value: nil}
)

func NewID(in string) Identifier {
	return Identifier{
		Name: in,
	}
}
