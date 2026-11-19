//go:build !windows

package persistencia

import (
	"errors"
	"os"
)

func bloquear(string) (*os.File, error) {
	return nil, errors.New("almacenamiento verificado solo en Windows")
}
func reemplazar(string, string) error { return errors.New("almacenamiento verificado solo en Windows") }
