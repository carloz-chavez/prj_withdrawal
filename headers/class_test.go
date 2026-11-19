package headers

import "testing"

func TestAbonoExitoso(t *testing.T) {
	cuenta := NewCuenta("Alice", 100.00)

	err := cuenta.Abonar(50.00)

	if err != nil {
		t.Fatalf("No se esperaba un error: %v", err)
	}

	if cuenta.ObtenerSaldo() != 150.00 {
		t.Errorf(
			"Se esperaba saldo 150.00, se obtuvo %.2f",
			cuenta.ObtenerSaldo(),
		)
	}
}

func TestAbonoNegativo(t *testing.T) {
	cuenta := NewCuenta("Alice", 100.00)

	err := cuenta.Abonar(-20.00)

	if err == nil {
		t.Error("Se esperaba un error por abono negativo")
	}

	if cuenta.ObtenerSaldo() != 100.00 {
		t.Error("El saldo no debe cambiar si el abono es inválido")
	}
}

func TestAbonoCero(t *testing.T) {
	cuenta := NewCuenta("Alice", 100.00)

	err := cuenta.Abonar(0.00)

	if err == nil {
		t.Error("Se esperaba un error por abono de cero")
	}

	if cuenta.ObtenerSaldo() != 100.00 {
		t.Error("El saldo no debe cambiar si el abono es cero")
	}
}
