package main

import (
	"flag"
	"fmt"
	"os"

	"prj_withdrawal/internal/cli"
)

func main() {
	ruta := flag.String("datos", "data/estado.json", "Ruta del estado JSON persistente (PEN)")
	legado := flag.Bool("memoria", false, "Usar el flujo anterior sin persistencia")
	flag.Parse()
	fmt.Println("=== SISTEMA BANCARIO INTERACTIVO ===")
	if !*legado {
		if err := cli.EjecutarPersistente(os.Stdin, os.Stdout, *ruta); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	var nombre string
	var saldoInicial float64

	fmt.Print("Ingrese el nombre del titular: ")
	fmt.Scanln(&nombre)

	fmt.Print("Ingrese el saldo inicial: ")
	if _, err := fmt.Scanln(&saldoInicial); err != nil {
		fmt.Fprintln(os.Stderr, "Saldo inicial inválido:", err)
		os.Exit(1)
	}

	if err := cli.Ejecutar(
		os.Stdin,
		os.Stdout,
		nombre,
		saldoInicial,
	); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
