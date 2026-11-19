// Package banco coordina las reglas y la confirmación de operaciones de US-04.
package banco

import (
	"errors"
	"time"
)

var (
	ErrPendiente    = errors.New("US-04 pendiente: contrato sin implementación")
	ErrPersistencia = errors.New("persistencia no confirmada")
	ErrIncierto     = errors.New("confirmación incierta: requiere recuperación")
	ErrCorrupto     = errors.New("estado corrupto")
	ErrEnUso        = errors.New("almacenamiento en uso")
	ErrConflicto    = errors.New("solicitud reutilizada con datos distintos")
)

type Solicitud struct {
	ID   string
	Tipo string
	// Entrada decimal original: no convertir mediante float64.
	Monto string
}

type Transaccion struct {
	ID          string    `json:"id"`
	SolicitudID string    `json:"solicitud_id"`
	CuentaID    string    `json:"cuenta_id"`
	Fecha       time.Time `json:"fecha"`
	Secuencia   int64     `json:"secuencia"`
	Tipo        string    `json:"tipo"`
	Entrada     string    `json:"entrada"`
	// nil para entradas sin representación monetaria válida.
	Monto          *int64 `json:"monto_centimos"`
	Moneda         string `json:"moneda"`
	Estado         string `json:"estado"`
	Motivo         string `json:"motivo,omitempty"`
	SaldoAnterior  int64  `json:"saldo_anterior"`
	SaldoPosterior int64  `json:"saldo_posterior"`
}

type Estado struct {
	Titular      string        `json:"titular"`
	Version      int           `json:"version"`
	CuentaID     string        `json:"cuenta_id"`
	Moneda       string        `json:"moneda"`
	SaldoInicial int64         `json:"saldo_inicial_centimos"`
	Saldo        int64         `json:"saldo_centimos"`
	Historial    []Transaccion `json:"historial"`
}

// Almacen confirma el documento completo. Un fallo incierto no equivale a rollback.
// Cargar devuelve os.ErrNotExist únicamente cuando no hay estado inicializado.
type Almacen interface {
	Cargar() (Estado, error)
	Confirmar(Estado) error
}

type Opciones struct {
	Titular      string
	CuentaID     string
	SaldoInicial int64
	Ahora        func() time.Time
	NuevoID      func() string
}

type Operador interface {
	Operar(Solicitud) (Transaccion, error)
	Estado() Estado
}
