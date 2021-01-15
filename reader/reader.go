package reader

import "github.com/coreos/pkg/dlopen"

import "C"

func ReadTypeInfo(from string) (string, error) {
	handle, err := dlopen.GetHandle([]string{from})
	if err != nil {
		return "", err
	}

	sym, err := handle.GetSymbolPointer("__tawa_types")
	if err != nil {
		return "", err
	}

	str := C.GoString((*C.char)(sym))
	return str, nil
}
