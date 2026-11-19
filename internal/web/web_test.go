package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"prj_withdrawal/internal/banco"
	"prj_withdrawal/internal/persistencia"
	"strings"
	"sync"
	"testing"
)

type operador struct{}

func (operador) Estado() banco.Estado {
	return banco.Estado{Titular: "Prueba", CuentaID: "cuenta", Moneda: "PEN", Saldo: 123}
}

func TestProteccionYLimites(t *testing.T) {
	s, _ := servicio(t)
	h := Handler(s)
	r := httptest.NewRequest("POST", "http://localhost/api/abonos", strings.NewReader(`{"solicitud_id":"x","monto":"1"}`))
	r.Header.Set("Origin", "http://otro")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("origen: %d", w.Code)
	}
	if w := peticion(h, "POST", "/api/abonos", `{"solicitud_id":"x","monto":"`+strings.Repeat("1", 5000)+`"}`); w.Code != 413 {
		t.Fatalf("tamaño: %d", w.Code)
	}
	for _, path := range []string{"/data/estado.json", "/web.go", "/api/desconocida"} {
		if w := peticion(h, "GET", path, ""); w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
func TestConcurrenciaYRechazos(t *testing.T) {
	s, _ := servicio(t)
	h := Handler(s)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := peticion(h, "POST", "/api/abonos", `{"solicitud_id":"igual","monto":"1"}`)
			if w.Code != 200 {
				t.Errorf("concurrencia: %d", w.Code)
			}
		}()
	}
	wg.Wait()
	if s.Estado().Saldo != 1100 || len(s.Estado().Historial) != 1 {
		t.Fatal("duplicación concurrente")
	}
	for i, m := range []string{"0", "-1", "texto", "NaN", "Infinity", "1.001", "92233720368547758.08", "999999"} {
		body, _ := json.Marshal(map[string]string{"solicitud_id": fmt.Sprintf("rechazo-%d", i), "monto": m})
		w := peticion(h, "POST", "/api/retiros", string(body))
		if w.Code != 422 {
			t.Errorf("%s: %d", m, w.Code)
		}
	}
	if len(s.Estado().Historial) != 9 {
		t.Fatal("rechazos no registrados")
	}
}
func TestInciertoHTTP(t *testing.T) {
	s, m := servicio(t)
	m.fallo = banco.ErrIncierto
	w := peticion(Handler(s), "POST", "/api/retiros", `{"solicitud_id":"x","monto":"1"}`)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "incierto") || s.Estado().Saldo != 1000 {
		t.Fatalf("incierto: %d %s", w.Code, w.Body.String())
	}
}
func TestPersistenciaRealYBloqueo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "estado.json")
	a, err := persistencia.Abrir(ruta)
	if err != nil {
		t.Fatal(err)
	}
	s, err := banco.Abrir(a, banco.Opciones{CuentaID: "c", Titular: "Prueba", SaldoInicial: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if w := peticion(Handler(s), "POST", "/api/abonos", `{"solicitud_id":"real","monto":"3.25"}`); w.Code != 200 {
		t.Fatalf("abono: %d %s", w.Code, w.Body.String())
	}
	if otro, err := persistencia.Abrir(ruta); err == nil {
		otro.Cerrar()
		t.Fatal("segundo escritor permitido")
	}
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	a, err = persistencia.Abrir(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Cerrar()
	s, err = banco.Abrir(a, banco.Opciones{CuentaID: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Estado().Saldo != 1325 || len(s.Estado().Historial) != 1 {
		t.Fatal("recuperación fallida")
	}
}

type memoria struct {
	estado banco.Estado
	existe bool
	fallo  error
}

func (m *memoria) Cargar() (banco.Estado, error) {
	if !m.existe {
		return banco.Estado{}, os.ErrNotExist
	}
	return m.estado, nil
}
func (m *memoria) Confirmar(e banco.Estado) error {
	if m.fallo != nil {
		return m.fallo
	}
	m.estado = e
	m.existe = true
	return nil
}
func servicio(t *testing.T) (banco.Operador, *memoria) {
	t.Helper()
	m := &memoria{}
	s, err := banco.Abrir(m, banco.Opciones{Titular: "Prueba", CuentaID: "c", SaldoInicial: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return s, m
}
func peticion(h http.Handler, metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, "http://localhost"+ruta, bytes.NewBufferString(cuerpo))
	if metodo == "POST" {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestOperacionesHTTP(t *testing.T) {
	s, _ := servicio(t)
	h := Handler(s)
	body := `{"solicitud_id":"uno","monto":"2.50"}`
	for i := 0; i < 2; i++ {
		w := peticion(h, "POST", "/api/retiros", body)
		if w.Code != 200 {
			t.Fatalf("retiro: %d %s", w.Code, w.Body.String())
		}
	}
	if s.Estado().Saldo != 750 || len(s.Estado().Historial) != 1 {
		t.Fatal("idempotencia")
	}
	if w := peticion(h, "POST", "/api/retiros", `{"solicitud_id":"uno","monto":"3"}`); w.Code != 409 {
		t.Fatalf("conflicto: %d", w.Code)
	}
	if w := peticion(h, "POST", "/api/abonos", `{"solicitud_id":"dos","monto":"NaN"}`); w.Code != 422 {
		t.Fatalf("rechazo: %d", w.Code)
	}
	w := peticion(h, "GET", "/api/transacciones", "")
	if w.Code != 200 {
		t.Fatalf("historial: %d", w.Code)
	}
	var tx []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tx); err != nil || len(tx) != 2 {
		t.Fatalf("historial: %s %v", w.Body.String(), err)
	}
	if tx[0]["saldo_posterior_centimos"] != "750" {
		t.Fatalf("saldo: %v", tx[0])
	}
}
func TestErroresHTTP(t *testing.T) {
	s, m := servicio(t)
	h := Handler(s)
	for _, c := range []struct {
		method, path, body string
		code               int
	}{{"GET", "/api/retiros", "", 405}, {"POST", "/api/abonos", "{", 400}, {"POST", "/api/abonos", `{"solicitud_id":"x","monto":3}`, 400}} {
		if w := peticion(h, c.method, c.path, c.body); w.Code != c.code {
			t.Errorf("%s %s: %d", c.method, c.path, w.Code)
		}
	}
	m.fallo = banco.ErrPersistencia
	w := peticion(h, "POST", "/api/abonos", `{"solicitud_id":"fallo","monto":"1"}`)
	if w.Code != 503 || s.Estado().Saldo != 1000 {
		t.Fatalf("fallo: %d", w.Code)
	}
}
func (operador) Operar(banco.Solicitud) (banco.Transaccion, error) { return banco.Transaccion{}, nil }

func TestConsultarCuenta(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(operador{}).ServeHTTP(w, httptest.NewRequest("GET", "/api/cuenta", nil))
	if w.Code != 200 {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v["saldo_centimos"] != "123" {
		t.Fatalf("saldo: %v", v["saldo_centimos"])
	}
}
