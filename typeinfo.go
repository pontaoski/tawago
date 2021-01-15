package main

import (
	"encoding/json"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/pontaoski/tawago/reader"
)

type typeInfo struct {
	Functions map[string]string `json:"functions"`
}

func registerTypeInfoWithModule(t typeInfo, m *ir.Module) {
	data, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	g := m.NewGlobalDef("__tawa_types", constant.NewCharArray(append(data, 0)))
	g.Immutable = true
}

func getTypeInfoFromFile(f string) (t typeInfo, err error) {
	data, err := reader.ReadTypeInfo(f)
	if err != nil {
		return typeInfo{}, err
	}

	err = json.Unmarshal([]byte(data), &t)
	return
}
