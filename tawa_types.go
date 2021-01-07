package main

import "github.com/llir/llvm/ir/types"

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

	Boolean = LLVMType{Type: &types.IntType{BitSize: 1, TypeName: "bool"}}
	Niets   = LLVMType{Type: &types.VoidType{TypeName: "niets"}}
)
