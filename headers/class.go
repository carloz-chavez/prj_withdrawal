
/*
source file: class.go
Cuenta: Representa la entidad de negocio descrita en el Feature Gherkin.
Date: nov-2026
Author: J.Medianero.A (adaptado a Go)
*/

package headers

import (
    "errors"
    "fmt"
)

// Cuenta representa la entidad de negocio descrita en el Feature Gherkin.
type Cuenta struct {
    saldo  float64
    nombre string
}

// NewCuenta es el constructor equivalente a Cuenta(nombre, saldoInicial).
func NewCuenta(nombre string, saldoInicial float64) *Cuenta {
    return &Cuenta{
        nombre: nombre,
        saldo:  saldoInicial,
    }
}

// ObtenerSaldo devuelve el saldo actual.
func (c *Cuenta) ObtenerSaldo() float64 {
    return c.saldo
}

// Nombre devuelve el nombre del titular (útil para logs).
func (c *Cuenta) Nombre() string {
    return c.nombre
}

/*
Retirar intenta retirar una cantidad.
Regla: No se puede retirar más de lo que hay en la cuenta.
*/
func (c *Cuenta) Retirar(cantidad float64) (bool, error) {
    if cantidad < 0 {
        return false, errors.New("no se pueden retirar cantidades negativas")
    }

    if cantidad > c.saldo {
        return false, nil // Escenario: Withdrawal declined
    }

    c.saldo -= cantidad
    return true, nil // Escenario: Successful withdrawal
}

// String implementa fmt.Stringer para depuración.
func (c *Cuenta) String() string {
    return fmt.Sprintf("Cuenta(%s, saldo=%.2f)", c.nombre, c.saldo)
}
