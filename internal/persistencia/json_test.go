package persistencia_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"prj_withdrawal/internal/banco"
	"prj_withdrawal/internal/persistencia"
)

func archivo(t *testing.T, ruta string) persistencia.Archivo {
	t.Helper()
	a, err := persistencia.Abrir(ruta)
	if err != nil {
		t.Fatalf("US-04: se esperaba abrir almacenamiento JSON: %v", err)
	}
	return a
}

func servicio(t *testing.T, a banco.Almacen, inicial int64, id string) banco.Operador {
	t.Helper()
	s, err := banco.Abrir(a, banco.Opciones{CuentaID: "cuenta-1", SaldoInicial: inicial,
		Ahora:   func() time.Time { return time.Date(2026, 11, 10, 15, 0, 0, 0, time.UTC) },
		NuevoID: func() string { return id },
	})
	if err != nil {
		t.Fatalf("se esperaba recuperar/inicializar servicio: %v", err)
	}
	return s
}

func TestJSONYRecuperacion(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a := archivo(t, ruta)
	defer a.Cerrar()
	s := servicio(t, a, 10000, "tx-1")
	q := banco.Solicitud{ID: "sol-1", Tipo: "ABONO", Monto: "25.00"}
	tx, err := s.Operar(q)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var e banco.Estado
	if err := json.Unmarshal(contenido, &e); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if e.Version != 1 || e.Moneda != "PEN" || e.CuentaID != "cuenta-1" || e.SaldoInicial != 10000 || e.Saldo != 12500 || len(e.Historial) != 1 || !reflect.DeepEqual(e.Historial[0], tx) {
		t.Fatalf("documento incompleto: %+v", e)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatal(err)
	}
	for _, campo := range []string{"version", "cuenta_id", "moneda", "saldo_inicial_centimos", "saldo_centimos", "historial"} {
		if _, ok := campos[campo]; !ok {
			t.Errorf("falta campo JSON %s", campo)
		}
	}
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	a2 := archivo(t, ruta)
	defer a2.Cerrar()
	// Un valor inicial diferente no debe sobrescribir el saldo persistido.
	s2 := servicio(t, a2, 99900, "tx-2")
	if !reflect.DeepEqual(s2.Estado(), e) {
		t.Fatal("reinicio perdió estado")
	}
	repetida, err := s2.Operar(q)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repetida, tx) || !reflect.DeepEqual(s2.Estado(), e) {
		t.Fatal("reinicio permitió duplicación")
	}
}

func TestArchivoCorrupto(t *testing.T) {
	for _, contenido := range []string{
		`{"version":`,
		`{"version":99,"cuenta_id":"cuenta-1","moneda":"PEN","saldo_inicial_centimos":10000,"saldo_centimos":10000,"historial":[]}`,
		`{"version":1,"cuenta_id":"cuenta-1","moneda":"PEN","saldo_inicial_centimos":10000,"saldo_centimos":9000,"historial":[]}`,
	} {
		t.Run(contenido, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "estado.json")
			if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
				t.Fatal(err)
			}
			a, err := persistencia.Abrir(ruta)
			if err == nil {
				defer a.Cerrar()
				_, err = a.Cargar()
			}
			if !errors.Is(err, banco.ErrCorrupto) {
				t.Fatalf("se esperaba ErrCorrupto, recibido: %v", err)
			}
			despues, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}
			if string(despues) != contenido {
				t.Fatal("se sobrescribió archivo corrupto")
			}
		})
	}
}

func TestExclusionEscritor(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a := archivo(t, ruta)
	defer a.Cerrar()
	b, err := persistencia.Abrir(ruta)
	if b != nil {
		defer b.Cerrar()
	}
	if !errors.Is(err, banco.ErrEnUso) {
		t.Fatalf("se esperaba exclusión: %v", err)
	}
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	c := archivo(t, ruta)
	defer c.Cerrar()
}

func TestErrorRutaNoEsArchivo(t *testing.T) {
	ruta := t.TempDir()
	a, err := persistencia.Abrir(ruta)
	if err == nil {
		defer a.Cerrar()
		err = a.Confirmar(banco.Estado{Version: 1, CuentaID: "cuenta-1", Moneda: "PEN", SaldoInicial: 10000, Saldo: 10000})
	}
	if !errors.Is(err, banco.ErrPersistencia) {
		t.Fatalf("se esperaba error de persistencia: %v", err)
	}
	if info, err := os.Stat(ruta); err != nil || !info.IsDir() {
		t.Fatal("se alteró el directorio destino")
	}
}
