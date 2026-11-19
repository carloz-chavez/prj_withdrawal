// Package persistencia contiene el contrato de almacenamiento JSON de US-04.
package persistencia

import "prj_withdrawal/internal/banco"

type Archivo interface {
	banco.Almacen
	Cerrar() error
}
