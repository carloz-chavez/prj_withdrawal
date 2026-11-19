package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"prj_withdrawal/internal/banco"
)

func TestCLIPersistenteYReinicio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	var salida bytes.Buffer
	if err := EjecutarPersistente(strings.NewReader("Alice\n100.00\n2\n30.00\n3\n25.00\n2\n120.00\n3\nabc\n4\n0\n"), &salida, ruta); err != nil {
		t.Fatal(err)
	}
	for _, texto := range []string{"70.00", "95.00", "fondos insuficientes", "monto inválido", "RETIRO", "ABONO", "EXITOSA", "RECHAZADA"} {
		if !strings.Contains(salida.String(), texto) {
			t.Errorf("falta %q: %s", texto, salida.String())
		}
	}
	salida.Reset()
	if err := EjecutarPersistente(strings.NewReader("1\n4\n0\n"), &salida, ruta); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida.String(), "95.00") || strings.Contains(salida.String(), "saldo inicial") {
		t.Fatalf("no recuperó saldo: %s", salida.String())
	}
	if strings.Count(salida.String(), " | ") != 24 {
		t.Fatalf("historial no tiene cuatro registros: %s", salida.String())
	}
}

type operadorFallido struct{}

func (operadorFallido) Estado() banco.Estado { return banco.Estado{Saldo: 10000} }
func (operadorFallido) Operar(banco.Solicitud) (banco.Transaccion, error) {
	return banco.Transaccion{}, banco.ErrIncierto
}
func TestCLINoAnunciaExitoAnteFallo(t *testing.T) {
	var salida bytes.Buffer
	err := ejecutarOperador(strings.NewReader("2\n30.00\n"), &salida, operadorFallido{}, func() (string, error) { return "sol-controlada", nil })
	if !errors.Is(err, banco.ErrIncierto) || strings.Contains(salida.String(), "exitosa") || !strings.Contains(salida.String(), "sol-controlada") {
		t.Fatalf("falso éxito o error oculto: %v %s", err, salida.String())
	}
}
