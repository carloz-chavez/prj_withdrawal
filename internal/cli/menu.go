package cli

import (
	"bufio"
	"io"
	"strings"
)

// MostrarMenu devuelve las opciones del sistema bancario.
func MostrarMenu() string {
	return `====================================
       SISTEMA BANCARIO - CODELAB
====================================
1. Consultar saldo
2. Retirar efectivo
3. Abonar efectivo
0. Salir
`
}

// LeerOpcion obtiene la opción ingresada por el usuario.
func LeerOpcion(entrada io.Reader) (string, error) {
	lector := bufio.NewReader(entrada)

	texto, err := lector.ReadString('\n')

	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(texto), nil
}
