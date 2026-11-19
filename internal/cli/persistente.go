package cli

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"prj_withdrawal/internal/banco"
	"prj_withdrawal/internal/persistencia"
)

func identificador() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func dinero(c int64) string { return fmt.Sprintf("%d.%02d", c/100, c%100) }

// EjecutarPersistente mantiene el flujo anterior Ejecutar como opción independiente.
func EjecutarPersistente(entrada io.Reader, salida io.Writer, ruta string) (retorno error) {
	a, err := persistencia.Abrir(ruta)
	if err != nil {
		return err
	}
	defer func() { retorno = errors.Join(retorno, a.Cerrar()) }()
	lector := bufio.NewReader(entrada)
	leer := func() (string, error) {
		s, err := lector.ReadString('\n')
		if err == io.EOF && s != "" {
			err = nil
		}
		return strings.TrimSpace(s), err
	}
	e, err := a.Cargar()
	var opciones banco.Opciones
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprint(salida, "Ingrese el nombre del titular: ")
		nombre, err := leer()
		if err != nil {
			return err
		}
		if nombre == "" {
			return errors.New("titular requerido")
		}
		fmt.Fprint(salida, "Ingrese el saldo inicial PEN: ")
		texto, err := leer()
		if err != nil {
			return err
		}
		saldo, err := banco.ParsearSaldo(texto)
		if err != nil {
			return err
		}
		id, err := identificador()
		if err != nil {
			return err
		}
		opciones = banco.Opciones{CuentaID: id, Titular: nombre, SaldoInicial: saldo}
	} else if err != nil {
		return err
	} else {
		opciones = banco.Opciones{CuentaID: e.CuentaID}
	}
	s, err := banco.Abrir(a, opciones)
	if err != nil {
		return err
	}
	return ejecutarOperador(lector, salida, s, identificador)
}

func ejecutarOperador(entrada io.Reader, salida io.Writer, s banco.Operador, nuevoID func() (string, error)) error {
	lector, ok := entrada.(*bufio.Reader)
	if !ok {
		lector = bufio.NewReader(entrada)
	}
	leer := func() (string, error) {
		linea, err := lector.ReadString('\n')
		if err == io.EOF && linea != "" {
			err = nil
		}
		return strings.TrimSpace(linea), err
	}
	for {
		fmt.Fprint(salida, "\n1. Consultar saldo\n2. Retirar efectivo\n3. Abonar efectivo\n4. Consultar historial\n0. Salir\nSeleccione una opción: ")
		opcion, err := leer()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch opcion {
		case "0":
			fmt.Fprintln(salida, "Saliendo del sistema...")
			return nil
		case "1":
			fmt.Fprintf(salida, "Saldo actual: %s PEN\n", dinero(s.Estado().Saldo))
		case "4":
			historial := s.Estado().Historial
			if len(historial) == 0 {
				fmt.Fprintln(salida, "Sin transacciones")
			}
			for _, tx := range historial {
				monto := tx.Entrada
				if tx.Monto != nil {
					monto = dinero(*tx.Monto)
				}
				fmt.Fprintf(salida, "%s | %s | %s | %s PEN | %s | Saldo: %s PEN | %s\n", tx.ID, tx.Fecha.Format("2006-01-02T15:04:05Z"), tx.Tipo, monto, tx.Estado, dinero(tx.SaldoPosterior), tx.Motivo)
			}
		case "2", "3":
			fmt.Fprint(salida, "Ingrese el monto PEN: ")
			monto, err := leer()
			if err != nil {
				return err
			}
			id, err := nuevoID()
			if err != nil {
				return err
			}
			tipo := "RETIRO"
			if opcion == "3" {
				tipo = "ABONO"
			}
			tx, err := s.Operar(banco.Solicitud{ID: id, Tipo: tipo, Monto: monto})
			if err != nil {
				fmt.Fprintf(salida, "Error de almacenamiento/operación (solicitud %s): %v\n", id, err)
				if errors.Is(err, banco.ErrIncierto) {
					return err
				}
				continue
			}
			if tx.Estado == "RECHAZADA" {
				fmt.Fprintf(salida, "Operación rechazada: %s. Saldo: %s PEN\n", tx.Motivo, dinero(tx.SaldoPosterior))
			} else {
				fmt.Fprintf(salida, "Operación exitosa. Nuevo saldo: %s PEN\n", dinero(tx.SaldoPosterior))
			}
		default:
			fmt.Fprintln(salida, "Opción inválida")
		}
	}
}
