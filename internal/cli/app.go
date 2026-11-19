package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"prj_withdrawal/headers"
)

// Ejecutar inicia el menú bancario interactivo.
func Ejecutar(entrada io.Reader, salida io.Writer, nombre string, saldoInicial float64) error {
	cuenta := headers.NewCuenta(nombre, saldoInicial)
	lector := bufio.NewReader(entrada)

	for {
		fmt.Fprint(salida, MostrarMenu())
		fmt.Fprint(salida, "Seleccione una opción: ")

		texto, err := lector.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if err == io.EOF && texto == "" {
			return nil
		}

		opcion := strings.TrimSpace(texto)

		switch opcion {
		case "1":
			fmt.Fprintf(salida, "Saldo actual: %.2f\n", cuenta.ObtenerSaldo())

		case "2":
			fmt.Fprint(salida, "Ingrese el monto a retirar: ")

			montoTexto, err := lector.ReadString('\n')
			if err != nil && err != io.EOF {
				return err
			}

			monto, err := strconv.ParseFloat(strings.TrimSpace(montoTexto), 64)
			if err != nil {
				fmt.Fprintln(salida, "Monto inválido")
				continue
			}

			exito, err := cuenta.Retirar(monto)
			if err != nil {
				fmt.Fprintf(salida, "Error: %v\n", err)
			} else if !exito {
				fmt.Fprintln(salida, "Saldo insuficiente")
			} else {
				fmt.Fprintf(salida, "Retiro exitoso. Nuevo saldo: %.2f\n", cuenta.ObtenerSaldo())
			}
		case "3":
			fmt.Fprint(salida, "Ingrese el monto a abonar: ")

			montoTexto, err := lector.ReadString('\n')
			if err != nil && err != io.EOF {
				return err
			}

			monto, err := strconv.ParseFloat(strings.TrimSpace(montoTexto), 64)
			if err != nil {
				fmt.Fprintln(salida, "Monto inválido")
				continue
			}

			err = cuenta.Abonar(monto)
			if err != nil {
				fmt.Fprintf(salida, "Error: %v\n", err)
			} else {
				fmt.Fprintf(
					salida,
					"Abono exitoso. Nuevo saldo: %.2f\n",
					cuenta.ObtenerSaldo(),
				)
			}
		case "0":
			fmt.Fprintln(salida, "Saliendo del sistema...")
			return nil

		default:
			fmt.Fprintln(salida, "Opción inválida")
		}

		fmt.Fprintln(salida)
	}
}
