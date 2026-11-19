package persistencia

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"prj_withdrawal/internal/banco"
)

func iniciar(t *testing.T, ruta string) (Archivo, banco.Operador) {
	t.Helper()
	a, err := Abrir(ruta)
	if err != nil {
		t.Fatal(err)
	}
	s, err := banco.Abrir(a, banco.Opciones{CuentaID: "cuenta-prueba", Titular: "Alice", SaldoInicial: 10000})
	if err != nil {
		a.Cerrar()
		t.Fatal(err)
	}
	return a, s
}

// El binario de pruebas actúa como proceso escritor independiente. Todos los
// caminos recibidos proceden de TempDir del proceso padre.
func TestProcesoAuxiliar(t *testing.T) {
	modo := os.Getenv("US04_MODO")
	if modo == "" {
		return
	}
	ruta := os.Getenv("US04_RUTA")
	a, err := Abrir(ruta)
	if modo == "exclusion" {
		if !errors.Is(err, banco.ErrEnUso) {
			os.Exit(5)
		}
		os.Exit(0)
	}
	if err != nil {
		os.Exit(6)
	}
	s, err := banco.Abrir(a, banco.Opciones{CuentaID: "cuenta-prueba"})
	if err != nil {
		os.Exit(7)
	}
	real := a.(*archivo)
	real.reemplazo = func(origen, destino string) error {
		if modo == "despues" {
			if err := reemplazar(origen, destino); err != nil {
				os.Exit(8)
			}
		}
		os.Exit(0) // sin Cerrar: simula terminación entre diario y respuesta al cliente
		return nil
	}
	_, _ = s.Operar(banco.Solicitud{ID: "sol-caida", Tipo: "ABONO", Monto: "25.00"})
	os.Exit(9)
}

func proceso(t *testing.T, ruta, modo string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcesoAuxiliar$")
	cmd.Env = append(os.Environ(), "US04_RUTA="+ruta, "US04_MODO="+modo)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("proceso %s: %v %s", modo, err, b)
	}
}

func TestExclusionEntreProcesos(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a, _ := iniciar(t, ruta)
	proceso(t, ruta, "exclusion")
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	b, err := Abrir(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
}

func TestRecuperacionTrasTerminacion(t *testing.T) {
	for _, modo := range []string{"antes", "despues"} {
		t.Run(modo, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "estado.json")
			a, _ := iniciar(t, ruta)
			a.Cerrar()
			proceso(t, ruta, modo)
			b, s := iniciar(t, ruta)
			defer b.Cerrar()
			if e := s.Estado(); e.Saldo != 12500 || len(e.Historial) != 1 || e.Titular != "Alice" {
				t.Fatalf("recuperación incorrecta: %+v", e)
			}
			if _, err := s.Operar(banco.Solicitud{ID: "sol-caida", Tipo: "ABONO", Monto: "25.00"}); err != nil {
				t.Fatal(err)
			}
			if len(s.Estado().Historial) != 1 {
				t.Fatal("reintento duplicó operación")
			}
		})
	}
}

func TestReemplazoFallidoYRecuperacion(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a, s := iniciar(t, ruta)
	a.(*archivo).reemplazo = func(string, string) error { return errors.New("fallo simulado de ReplaceFileW") }
	q := banco.Solicitud{ID: "sol-fallo", Tipo: "RETIRO", Monto: "30.00"}
	if _, err := s.Operar(q); !errors.Is(err, banco.ErrIncierto) {
		t.Fatalf("se esperaba incertidumbre: %v", err)
	}
	if s.Estado().Saldo != 10000 {
		t.Fatal("publicó éxito antes de reemplazar")
	}
	a.Cerrar()
	b, s2 := iniciar(t, ruta)
	defer b.Cerrar()
	if s2.Estado().Saldo != 7000 {
		t.Fatal("diario confirmado no recuperado")
	}
	if _, err := s2.Operar(q); err != nil {
		t.Fatal(err)
	}
	if len(s2.Estado().Historial) != 1 {
		t.Fatal("duplicación tras recuperación")
	}
}

func TestDiarioParcialBloquea(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a, _ := iniciar(t, ruta)
	a.Cerrar()
	f, err := os.OpenFile(ruta+".journal", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(`{"estado":`); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if a, err := Abrir(ruta); !errors.Is(err, banco.ErrIncierto) {
		if a != nil {
			a.Cerrar()
		}
		t.Fatalf("debe bloquear diario parcial: %v", err)
	}
}

func TestFalloDeEscrituraDiario(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a, err := Abrir(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Cerrar()
	if err := os.Mkdir(ruta+".journal", 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := banco.Abrir(a, banco.Opciones{CuentaID: "cuenta-prueba", SaldoInicial: 10000}); err == nil {
		t.Fatal("almacén no escribible aceptado")
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Fatal("creó estado sin diario confirmado")
	}
}
