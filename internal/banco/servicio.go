package banco

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

type servicio struct {
	mu       sync.Mutex
	almacen  Almacen
	estado   Estado
	opciones Opciones
	incierto bool
}

// Abrir recupera el documento confirmado o confirma un estado inicial nuevo.
// El llamador debe proporcionar un almacén exclusivo para este operador.
func Abrir(a Almacen, o Opciones) (Operador, error) {
	if a == nil || strings.TrimSpace(o.CuentaID) == "" {
		return nil, errors.New("almacén e identidad de cuenta requeridos")
	}
	if o.Ahora == nil {
		o.Ahora = time.Now
	}
	e, err := a.Cargar()
	if errors.Is(err, os.ErrNotExist) {
		if o.SaldoInicial < 0 {
			return nil, errors.New("saldo inicial negativo")
		}
		e = Estado{Version: 1, CuentaID: o.CuentaID, Titular: o.Titular, Moneda: "PEN", SaldoInicial: o.SaldoInicial, Saldo: o.SaldoInicial}
		if err := a.Confirmar(copiaEstado(e)); err != nil {
			return nil, falloConfirmacion(err)
		}
	} else if err != nil {
		return nil, err
	}
	if err := validarEstado(e); err != nil {
		return nil, err
	}
	if e.CuentaID != o.CuentaID {
		return nil, errors.New("la cuenta no corresponde al estado almacenado")
	}
	return &servicio{almacen: a, estado: copiaEstado(e), opciones: o}, nil
}

func (s *servicio) Estado() Estado {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copiaEstado(s.estado)
}

func (s *servicio) Operar(q Solicitud) (Transaccion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.incierto {
		return Transaccion{}, ErrIncierto
	}
	if strings.TrimSpace(q.ID) == "" {
		return Transaccion{}, errors.New("identificador de solicitud requerido")
	}
	for _, tx := range s.estado.Historial {
		if tx.SolicitudID == q.ID {
			if tx.Tipo != q.Tipo || tx.Entrada != q.Monto {
				return Transaccion{}, ErrConflicto
			}
			return copiaTransaccion(tx), nil
		}
	}
	if q.Tipo != "RETIRO" && q.Tipo != "ABONO" {
		return Transaccion{}, errors.New("tipo de operación desconocido")
	}
	id, err := s.nuevoID()
	if err != nil {
		return Transaccion{}, err
	}
	tx := Transaccion{ID: id, SolicitudID: q.ID, CuentaID: s.estado.CuentaID,
		Fecha: s.opciones.Ahora().UTC(), Secuencia: int64(len(s.estado.Historial)) + 1,
		Tipo: q.Tipo, Entrada: q.Monto, Moneda: "PEN", Estado: "RECHAZADA",
		SaldoAnterior: s.estado.Saldo, SaldoPosterior: s.estado.Saldo}
	monto, valido := centimos(q.Monto)
	switch {
	case !valido:
		tx.Motivo = "monto inválido"
	default:
		tx.Monto = &monto
		switch {
		case q.Tipo == "RETIRO" && monto > s.estado.Saldo:
			tx.Motivo = "fondos insuficientes"
		case q.Tipo == "ABONO" && monto > math.MaxInt64-s.estado.Saldo:
			tx.Motivo = "desbordamiento de saldo"
		default:
			tx.Estado = "EXITOSA"
			if q.Tipo == "RETIRO" {
				tx.SaldoPosterior -= monto
			} else {
				tx.SaldoPosterior += monto
			}
		}
	}
	candidato := copiaEstado(s.estado)
	candidato.Saldo = tx.SaldoPosterior
	candidato.Historial = append(candidato.Historial, copiaTransaccion(tx))
	if err := s.almacen.Confirmar(copiaEstado(candidato)); err != nil {
		err = falloConfirmacion(err)
		s.incierto = errors.Is(err, ErrIncierto)
		return Transaccion{}, err
	}
	s.estado = candidato
	return copiaTransaccion(tx), nil
}

func falloConfirmacion(err error) error {
	if errors.Is(err, ErrIncierto) {
		return err
	}
	if errors.Is(err, ErrPersistencia) {
		return err
	}
	// Solo el adaptador puede asegurar que un fallo no tuvo efecto persistente.
	return fmt.Errorf("%w: %w", ErrIncierto, err)
}

func (s *servicio) nuevoID() (string, error) {
	var id string
	if s.opciones.NuevoID != nil {
		id = s.opciones.NuevoID()
	} else {
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return "", err
		}
		id = hex.EncodeToString(bytes[:])
	}
	if strings.TrimSpace(id) == "" {
		return "", errors.New("identificador de transacción vacío")
	}
	for _, tx := range s.estado.Historial {
		if tx.ID == id {
			return "", errors.New("identificador de transacción duplicado")
		}
	}
	return id, nil
}

// centimos admite únicamente dígitos y un punto con uno o dos decimales.
// Acumula directamente céntimos comprobando desbordamiento en cada paso.
func centimos(texto string) (int64, bool) {
	partes := strings.Split(texto, ".")
	if len(partes) > 2 || partes[0] == "" {
		return 0, false
	}
	digitos := partes[0]
	if len(partes) == 2 {
		if len(partes[1]) < 1 || len(partes[1]) > 2 {
			return 0, false
		}
		digitos += partes[1]
		if len(partes[1]) == 1 {
			digitos += "0"
		}
	} else {
		digitos += "00"
	}
	var resultado int64
	for _, r := range digitos {
		if r < '0' || r > '9' {
			return 0, false
		}
		d := int64(r - '0')
		if resultado > (math.MaxInt64-d)/10 {
			return 0, false
		}
		resultado = resultado*10 + d
	}
	return resultado, resultado > 0
}

func copiaTransaccion(tx Transaccion) Transaccion {
	if tx.Monto != nil {
		m := *tx.Monto
		tx.Monto = &m
	}
	return tx
}

func copiaEstado(e Estado) Estado {
	if e.Historial != nil {
		h := make([]Transaccion, len(e.Historial))
		for i, tx := range e.Historial {
			h[i] = copiaTransaccion(tx)
		}
		e.Historial = h
	}
	return e
}

// validarEstado comprueba invariantes de dominio; no sustituye la recuperación
// de archivos o la validación de escrituras que corresponden al adaptador.
func validarEstado(e Estado) error {
	if e.Version != 1 || e.Moneda != "PEN" || e.CuentaID == "" || e.SaldoInicial < 0 || e.Saldo < 0 {
		return ErrCorrupto
	}
	saldo := e.SaldoInicial
	ids, solicitudes := map[string]bool{}, map[string]bool{}
	for i, tx := range e.Historial {
		if tx.ID == "" || tx.SolicitudID == "" || ids[tx.ID] || solicitudes[tx.SolicitudID] || tx.CuentaID != e.CuentaID || tx.Moneda != "PEN" || tx.Secuencia != int64(i)+1 || tx.Fecha.IsZero() || tx.SaldoAnterior != saldo {
			return ErrCorrupto
		}
		if _, offset := tx.Fecha.Zone(); offset != 0 {
			return ErrCorrupto
		}
		ids[tx.ID], solicitudes[tx.SolicitudID] = true, true
		if tx.Tipo != "RETIRO" && tx.Tipo != "ABONO" {
			return ErrCorrupto
		}
		m, ok := centimos(tx.Entrada)
		if ok {
			if tx.Monto == nil || *tx.Monto != m {
				return ErrCorrupto
			}
		} else if tx.Monto != nil {
			return ErrCorrupto
		}
		switch tx.Estado {
		case "EXITOSA":
			if !ok || tx.Motivo != "" {
				return ErrCorrupto
			}
			if tx.Tipo == "RETIRO" {
				if m > saldo {
					return ErrCorrupto
				}
				saldo -= m
			} else {
				if m > math.MaxInt64-saldo {
					return ErrCorrupto
				}
				saldo += m
			}
		case "RECHAZADA":
			esperado := ""
			switch {
			case !ok:
				esperado = "monto inválido"
			case tx.Tipo == "RETIRO" && m > saldo:
				esperado = "fondos insuficientes"
			case tx.Tipo == "ABONO" && m > math.MaxInt64-saldo:
				esperado = "desbordamiento de saldo"
			}
			if esperado == "" || tx.Motivo != esperado {
				return ErrCorrupto
			}
		default:
			return ErrCorrupto
		}
		if tx.SaldoPosterior != saldo {
			return ErrCorrupto
		}
	}
	if saldo != e.Saldo {
		return ErrCorrupto
	}
	return nil
}

// ValidarEstado permite al adaptador comprobar las mismas invariantes de dominio.
func ValidarEstado(e Estado) error { return validarEstado(e) }

// ParsearSaldo interpreta un saldo inicial decimal exacto, incluido cero.
func ParsearSaldo(s string) (int64, error) {
	if s == "0" || s == "0.0" || s == "0.00" {
		return 0, nil
	}
	if m, ok := centimos(s); ok {
		return m, nil
	}
	return 0, errors.New("saldo inicial inválido")
}
