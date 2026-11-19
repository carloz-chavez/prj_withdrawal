package banco_test

import (
	"errors"
	"testing"

	"prj_withdrawal/internal/banco"
)

func TestCopiasDefensivas(t *testing.T) {
	m := &memoria{}
	s := abrir(t, m)
	tx, err := s.Operar(banco.Solicitud{ID: "sol-1", Tipo: "ABONO", Monto: "1.00"})
	if err != nil {
		t.Fatal(err)
	}
	*tx.Monto = 999
	e := s.Estado()
	e.Saldo = 0
	e.Historial[0].ID = "alterado"
	*e.Historial[0].Monto = 888
	real := s.Estado()
	if real.Saldo != 10100 || real.Historial[0].ID == "alterado" || *real.Historial[0].Monto != 100 {
		t.Fatal("una copia expone estado mutable del servicio")
	}
	guardado, err := m.Cargar()
	if err != nil || *guardado.Historial[0].Monto != 100 {
		t.Fatal("la respuesta expone estado del almacén")
	}
}

func TestReintentoInvalidoTrasRecuperar(t *testing.T) {
	m := &memoria{}
	s := abrir(t, m)
	q := banco.Solicitud{ID: "invalida", Tipo: "ABONO", Monto: "NaN"}
	tx, err := s.Operar(q)
	if err != nil || tx.Estado != "RECHAZADA" {
		t.Fatalf("rechazo no confirmado: %+v %v", tx, err)
	}
	s = abrir(t, m)
	repetida, err := s.Operar(q)
	if err != nil || repetida.ID != tx.ID || len(s.Estado().Historial) != 1 {
		t.Fatal("rechazo duplicado tras recuperación")
	}
	q.Monto = "abc"
	if _, err := s.Operar(q); !errors.Is(err, banco.ErrConflicto) {
		t.Fatalf("entradas inválidas diferentes deben producir conflicto: %v", err)
	}
}

func TestFalloNoClasificadoBloqueaServicio(t *testing.T) {
	m := &memoria{}
	s := abrir(t, m)
	m.fallo = errors.New("respuesta del adaptador desconocida")
	q := banco.Solicitud{ID: "sol-1", Tipo: "RETIRO", Monto: "1.00"}
	if _, err := s.Operar(q); !errors.Is(err, banco.ErrIncierto) {
		t.Fatalf("fallo desconocido no permite asegurar rollback: %v", err)
	}
	m.fallo = nil
	if _, err := s.Operar(q); !errors.Is(err, banco.ErrIncierto) {
		t.Fatalf("debe exigir recuperación antes del reintento: %v", err)
	}
	if s.Estado().Saldo != 10000 {
		t.Fatal("se publicó un saldo no confirmado")
	}
}
