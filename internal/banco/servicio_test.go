package banco_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"prj_withdrawal/internal/banco"
)

// Doble de la frontera de almacenamiento; no implementa reglas bancarias.
type memoria struct {
	estado                 banco.Estado
	existe                 bool
	fallo                  error
	confirmarAntesDeFallar bool
}

func clonar(e banco.Estado) banco.Estado {
	e.Historial = append([]banco.Transaccion(nil), e.Historial...)
	for i := range e.Historial {
		if e.Historial[i].Monto != nil {
			m := *e.Historial[i].Monto
			e.Historial[i].Monto = &m
		}
	}
	return e
}
func (m *memoria) Cargar() (banco.Estado, error) {
	if !m.existe {
		return banco.Estado{}, os.ErrNotExist
	}
	return clonar(m.estado), nil
}
func (m *memoria) Confirmar(e banco.Estado) error {
	if m.fallo != nil && !m.confirmarAntesDeFallar {
		return m.fallo
	}
	m.estado, m.existe = clonar(e), true
	return m.fallo
}

var fecha = time.Date(2026, 11, 10, 15, 0, 0, 0, time.UTC)

func abrir(t *testing.T, m banco.Almacen) banco.Operador {
	t.Helper()
	var mu sync.Mutex
	n := 0
	s, err := banco.Abrir(m, banco.Opciones{
		CuentaID: "cuenta-1", SaldoInicial: 10000,
		Ahora:   func() time.Time { return fecha },
		NuevoID: func() string { mu.Lock(); defer mu.Unlock(); n++; return fmt.Sprintf("tx-%d", n) },
	})
	if err != nil {
		t.Fatalf("US-04: se esperaba abrir el servicio: %v", err)
	}
	return s
}

func TestOperaciones(t *testing.T) {
	for _, c := range []struct {
		nombre, tipo, monto, estado string
		saldo                       int64
	}{
		{"retiro", "RETIRO", "30.00", "EXITOSA", 7000},
		{"abono", "ABONO", "25.00", "EXITOSA", 12500},
		{"fondos_insuficientes", "RETIRO", "120.00", "RECHAZADA", 10000},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			m := &memoria{}
			s := abrir(t, m)
			tx, err := s.Operar(banco.Solicitud{ID: "sol-1", Tipo: c.tipo, Monto: c.monto})
			if err != nil {
				t.Fatal(err)
			}
			if tx.Estado != c.estado || tx.SaldoAnterior != 10000 || tx.SaldoPosterior != c.saldo {
				t.Fatalf("resultado incorrecto: %+v", tx)
			}
			if tx.ID == "" || tx.SolicitudID != "sol-1" || tx.CuentaID != "cuenta-1" || !tx.Fecha.Equal(fecha) || tx.Moneda != "PEN" || tx.Secuencia != 1 || tx.Monto == nil {
				t.Fatalf("registro incompleto: %+v", tx)
			}
			if c.estado == "RECHAZADA" && tx.Motivo == "" {
				t.Fatal("falta motivo")
			}
			e, err := m.Cargar()
			if err != nil {
				t.Fatal(err)
			}
			if e.Saldo != c.saldo || len(e.Historial) != 1 || !reflect.DeepEqual(e.Historial[0], tx) || !reflect.DeepEqual(e, s.Estado()) {
				t.Fatalf("saldo e historial no confirmados juntos: %+v", e)
			}
		})
	}
}

func TestMontosInvalidos(t *testing.T) {
	for _, tipo := range []string{"RETIRO", "ABONO"} {
		for _, monto := range []string{"0", "-1.00", "NaN", "+Inf", "-Inf", "abc", "", "1.001", "92233720368547758.08"} {
			t.Run(tipo+"/"+monto, func(t *testing.T) {
				m := &memoria{}
				s := abrir(t, m)
				tx, err := s.Operar(banco.Solicitud{ID: "sol-invalida", Tipo: tipo, Monto: monto})
				if err != nil {
					t.Fatal(err)
				}
				if tx.Estado != "RECHAZADA" || tx.Motivo == "" || tx.SaldoAnterior != 10000 || tx.SaldoPosterior != 10000 {
					t.Fatalf("rechazo incorrecto: %+v", tx)
				}
				e, err := m.Cargar()
				if err != nil {
					t.Fatal(err)
				}
				if e.Saldo != 10000 || len(e.Historial) != 1 || !reflect.DeepEqual(e.Historial[0], tx) {
					t.Fatal("falta rechazo persistido sin movimiento")
				}
			})
		}
	}
}

func TestFalloAlmacenamiento(t *testing.T) {
	for _, monto := range []string{"30.00", "120.00"} {
		t.Run(monto, func(t *testing.T) {
			m := &memoria{}
			s := abrir(t, m)
			antes := clonar(s.Estado())
			m.fallo = banco.ErrPersistencia
			_, err := s.Operar(banco.Solicitud{ID: "sol-fallo", Tipo: "RETIRO", Monto: monto})
			if !errors.Is(err, banco.ErrPersistencia) {
				t.Fatalf("se esperaba error de persistencia: %v", err)
			}
			e, err := m.Cargar()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(antes, s.Estado()) || !reflect.DeepEqual(antes, e) {
				t.Fatal("modificación parcial ante fallo")
			}
		})
	}
}

func TestReintentoSinDuplicacion(t *testing.T) {
	m := &memoria{}
	s := abrir(t, m)
	q := banco.Solicitud{ID: "sol-1", Tipo: "RETIRO", Monto: "30.00"}
	primera, err := s.Operar(q)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := s.Operar(q)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(primera, segunda) || s.Estado().Saldo != 7000 || len(s.Estado().Historial) != 1 {
		t.Fatal("reintento duplicado")
	}
	q.Monto = "40.00"
	if _, err := s.Operar(q); !errors.Is(err, banco.ErrConflicto) {
		t.Fatalf("se esperaba conflicto: %v", err)
	}
	if s.Estado().Saldo != 7000 || len(s.Estado().Historial) != 1 {
		t.Fatal("conflicto modificó estado")
	}
}

func TestConcurrencia(t *testing.T) {
	m := &memoria{}
	s := abrir(t, m)
	inicio := make(chan struct{})
	type resultado struct {
		tx  banco.Transaccion
		err error
	}
	resultados := make(chan resultado, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			<-inicio
			tx, err := s.Operar(banco.Solicitud{ID: fmt.Sprintf("sol-%d", i), Tipo: "RETIRO", Monto: "80.00"})
			resultados <- resultado{tx, err}
		}(i)
	}
	close(inicio)
	exitos, rechazos := 0, 0
	for i := 0; i < 2; i++ {
		select {
		case r := <-resultados:
			if r.err != nil {
				t.Fatal(r.err)
			}
			switch r.tx.Estado {
			case "EXITOSA":
				exitos++
			case "RECHAZADA":
				rechazos++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("operaciones bloqueadas")
		}
	}
	e := s.Estado()
	if exitos != 1 || rechazos != 1 || e.Saldo != 2000 || len(e.Historial) != 2 {
		t.Fatalf("invariante concurrente incumplida: %+v", e)
	}
	if e.Historial[0].ID == e.Historial[1].ID || e.Historial[0].Secuencia != 1 || e.Historial[1].Secuencia != 2 {
		t.Fatal("identidad o secuencia inválida")
	}
	guardado, err := m.Cargar()
	if err != nil || !reflect.DeepEqual(e, guardado) {
		t.Fatal("estado concurrente no persistido")
	}
}

func TestConfirmacionIncierta(t *testing.T) {
	for _, aplicado := range []bool{false, true} {
		t.Run(fmt.Sprint(aplicado), func(t *testing.T) {
			m := &memoria{}
			s := abrir(t, m)
			m.fallo, m.confirmarAntesDeFallar = banco.ErrIncierto, aplicado
			q := banco.Solicitud{ID: "sol-incierta", Tipo: "RETIRO", Monto: "30.00"}
			if _, err := s.Operar(q); !errors.Is(err, banco.ErrIncierto) {
				t.Fatalf("se esperaba resultado incierto: %v", err)
			}
			if _, err := s.Operar(banco.Solicitud{ID: "otra", Tipo: "ABONO", Monto: "1.00"}); !errors.Is(err, banco.ErrIncierto) {
				t.Fatalf("servicio debe bloquear nuevas operaciones: %v", err)
			}
			m.fallo = nil
			recuperado := abrir(t, m)
			if _, err := recuperado.Operar(q); err != nil {
				t.Fatal(err)
			}
			if recuperado.Estado().Saldo != 7000 || len(recuperado.Estado().Historial) != 1 {
				t.Fatal("recuperación duplicó o perdió operación")
			}
		})
	}
}

func TestCentimosExactosYDesbordamiento(t *testing.T) {
	t.Run("centimos_exactos", func(t *testing.T) {
		m := &memoria{}
		s := abrir(t, m)
		for i, monto := range []string{"0.10", "0.20"} {
			tx, err := s.Operar(banco.Solicitud{ID: fmt.Sprintf("decimal-%d", i), Tipo: "ABONO", Monto: monto})
			if err != nil || tx.Estado != "EXITOSA" {
				t.Fatalf("abono decimal rechazado: %+v, %v", tx, err)
			}
		}
		if s.Estado().Saldo != 10030 {
			t.Fatalf("saldo inexacto: %d", s.Estado().Saldo)
		}
	})
	t.Run("desbordamiento_saldo", func(t *testing.T) {
		const maximo int64 = 9223372036854775807
		m := &memoria{existe: true, estado: banco.Estado{Version: 1, CuentaID: "cuenta-1", Moneda: "PEN", SaldoInicial: maximo, Saldo: maximo}}
		s := abrir(t, m)
		tx, err := s.Operar(banco.Solicitud{ID: "overflow", Tipo: "ABONO", Monto: "0.01"})
		if err != nil || tx.Estado != "RECHAZADA" || tx.Motivo == "" {
			t.Fatalf("debe rechazar desbordamiento: %+v, %v", tx, err)
		}
		if s.Estado().Saldo != maximo || len(s.Estado().Historial) != 1 {
			t.Fatal("desbordamiento modificó saldo o perdió rechazo")
		}
	})
}
