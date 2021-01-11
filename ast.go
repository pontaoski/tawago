package main

type Kind int64
type Identifier struct {
	Name string
	Pos  Span
}
type Type interface {
	is_Type()
}
type Ident Identifier

func (v Ident) is_Type() {}

type FunctionPointer struct {
	Arguments []Type
	Returns   *Type
}

func (v FunctionPointer) is_Type() {}

type Struct []struct {
	Ident string
	Kind  Type
}

func (v Struct) is_Type() {}

type Literal interface {
	is_Literal()
}
type Integer int64

func (v Integer) is_Literal() {}

type StructLiteral struct {
	Ident  Identifier
	Fields map[string]Expression
}

func (v StructLiteral) is_Literal() {}

type StringLiteral string

func (v StringLiteral) is_Literal() {}

type Expression interface {
	is_Expression()
}
type Lit struct {
	Literal
}

func (v Lit) is_Expression() {}

type Var Identifier

func (v Var) is_Expression() {}

type Declaration struct {
	To    Identifier
	Value Expression
}

func (v Declaration) is_Expression() {}

type Field struct {
	Of    Expression
	Ident Identifier
}

func (v Field) is_Expression() {}

type MutDeclaration struct {
	To    Identifier
	Value Expression
}

func (v MutDeclaration) is_Expression() {}

type Assignment struct {
	To    Identifier
	Value Expression
	Pos   Span
}

func (v Assignment) is_Expression() {}

type FieldAssignment struct {
	Struct Expression
	Field  Identifier
	Value  Expression
	Pos    Span
}

func (v FieldAssignment) is_Expression() {}

type Call struct {
	Function  Identifier
	Arguments []Expression
}

func (v Call) is_Expression() {}

type Block []Expression

func (v Block) is_Expression() {}

type If struct {
	Condition Expression
	Then      Expression
	Else      Expression
}

func (v If) is_Expression() {}

type TopLevel interface {
	is_TopLevel()
}
type Func struct {
	Ident     Identifier
	Arguments []struct {
		Ident Identifier
		Kind  Type
	}

	Returns *Type
	Expr    Expression
}

func (v Func) is_TopLevel() {}

type Import string

func (v Import) is_TopLevel() {}

type TypeDeclaration struct {
	Ident Identifier
	Kind  Type
}

func (v TypeDeclaration) is_TopLevel() {}

type ASTNode interface {
	is_ASTNode()
}
type T struct {
	TopLevel
}

func (v T) is_ASTNode() {}

type E struct {
	Expression
}

func (v E) is_ASTNode() {}
