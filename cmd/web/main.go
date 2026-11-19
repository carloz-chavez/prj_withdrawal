package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"prj_withdrawal/internal/banco"
	"prj_withdrawal/internal/persistencia"
	"prj_withdrawal/internal/web"
)

func run() (ret error) {
	ruta := flag.String("datos", "data/estado.json", "Ruta del estado JSON")
	direccion := flag.String("escucha", "127.0.0.1:8080", "Dirección de escucha")
	titular := flag.String("titular", "Cuenta de demostración", "Titular para una cuenta nueva")
	inicial := flag.String("saldo-inicial", "0.00", "Saldo inicial en PEN para una cuenta nueva")
	flag.Parse()
	a, err := persistencia.Abrir(*ruta)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el almacenamiento (puede estar en uso): %w", err)
	}
	defer func() { ret = errors.Join(ret, a.Cerrar()) }()
	e, err := a.Cargar()
	o := banco.Opciones{}
	if errors.Is(err, os.ErrNotExist) {
		if strings.TrimSpace(*titular) == "" {
			return errors.New("titular requerido")
		}
		saldo, err := banco.ParsearSaldo(*inicial)
		if err != nil {
			return err
		}
		var id [16]byte
		if _, err = rand.Read(id[:]); err != nil {
			return err
		}
		o = banco.Opciones{CuentaID: hex.EncodeToString(id[:]), Titular: *titular, SaldoInicial: saldo}
	} else if err != nil {
		return err
	} else {
		o.CuentaID = e.CuentaID
	}
	s, err := banco.Abrir(a, o)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", *direccion)
	if err != nil {
		return err
	}
	servidor := &http.Server{Handler: web.Handler(s), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- servidor.Serve(ln) }()
	log.Printf("Aplicación disponible en http://%s", ln.Addr())
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		cierre, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := servidor.Shutdown(cierre); err != nil {
			_ = servidor.Close()
			return err
		}
		err = <-done
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
