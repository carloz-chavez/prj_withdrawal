package persistencia

import (
	"os"
	"syscall"
	"unsafe"
)

var kernel = syscall.NewLazyDLL("kernel32.dll")
var replaceFile = kernel.NewProc("ReplaceFileW")

func bloquear(ruta string) (*os.File, error) {
	p, err := syscall.UTF16PtrFromString(ruta)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), ruta), nil
}

// No se atribuye atomicidad/durabilidad al reemplazo: el diario decide recuperación.
func reemplazar(origen, destino string) error {
	if _, err := os.Stat(destino); os.IsNotExist(err) {
		return os.Rename(origen, destino)
	}
	d, err := syscall.UTF16PtrFromString(destino)
	if err != nil {
		return err
	}
	o, err := syscall.UTF16PtrFromString(origen)
	if err != nil {
		return err
	}
	r, _, e := replaceFile.Call(uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(o)), 0, 0, 0, 0)
	if r == 0 {
		return e
	}
	return nil
}
