package cli

import (
	"strings"
	"testing"
)

func TestMostrarMenu(t *testing.T) {
	menu := MostrarMenu()

	opciones := []string{
		"1. Consultar saldo",
		"2. Retirar efectivo",
		"0. Salir",
	}

	for _, opcion := range opciones {
		if !strings.Contains(menu, opcion) {
			t.Errorf(
				"El menú debe contener %q",
				opcion,
			)
		}
	}
}

func TestLeerOpcion(t *testing.T) {
	entrada := strings.NewReader("2\n")

	opcion, err := LeerOpcion(entrada)

	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if opcion != "2" {
		t.Errorf("Se esperaba la opción 2, se obtuvo %q", opcion)
	}
}
