package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIConsultarSaldo(t *testing.T) {
	entrada := strings.NewReader("1\n0\n")
	var salida bytes.Buffer

	err := Ejecutar(entrada, &salida, "Alice", 234.56)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if !strings.Contains(salida.String(), "234.56") {
		t.Errorf("Debe mostrar el saldo inicial")
	}
}

func TestCLIRetiroExitoso(t *testing.T) {
	entrada := strings.NewReader("2\n200\n1\n0\n")
	var salida bytes.Buffer

	err := Ejecutar(entrada, &salida, "Alice", 234.56)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if !strings.Contains(salida.String(), "34.56") {
		t.Errorf("Debe mostrar el saldo restante de 34.56")
	}
}

func TestCLIFondosInsuficientes(t *testing.T) {
	entrada := strings.NewReader("2\n200\n1\n0\n")
	var salida bytes.Buffer

	err := Ejecutar(entrada, &salida, "Hamza", 198.76)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if !strings.Contains(salida.String(), "Saldo insuficiente") {
		t.Errorf("Debe rechazar el retiro por fondos insuficientes")
	}

	if !strings.Contains(salida.String(), "198.76") {
		t.Errorf("El saldo debe permanecer intacto")
	}
}

func TestCLIAbonoExitoso(t *testing.T) {
	entrada := strings.NewReader("3\n50\n1\n0\n")
	var salida bytes.Buffer

	err := Ejecutar(entrada, &salida, "Alice", 100.00)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if !strings.Contains(salida.String(), "Abono exitoso") {
		t.Error("Debe confirmar el abono exitoso")
	}

	if !strings.Contains(salida.String(), "150.00") {
		t.Error("El saldo esperado es 150.00")
	}
}

func TestCLIAbonoInvalido(t *testing.T) {
	entrada := strings.NewReader("3\n-20\n1\n0\n")
	var salida bytes.Buffer

	err := Ejecutar(entrada, &salida, "Alice", 100.00)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if !strings.Contains(salida.String(), "Error:") {
		t.Error("Debe mostrar un error por abono negativo")
	}

	if !strings.Contains(salida.String(), "100.00") {
		t.Error("El saldo debe permanecer intacto")
	}
}
