package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"prj_withdrawal/internal/banco"
)

//go:embed index.html app.js style.css
var assets embed.FS

type transaccion struct {
	ID          string  `json:"id"`
	SolicitudID string  `json:"solicitud_id"`
	Fecha       string  `json:"fecha"`
	Tipo        string  `json:"tipo"`
	Entrada     string  `json:"entrada"`
	Monto       *string `json:"monto_centimos"`
	Estado      string  `json:"estado"`
	Motivo      string  `json:"motivo"`
	Anterior    string  `json:"saldo_anterior_centimos"`
	Posterior   string  `json:"saldo_posterior_centimos"`
}

func serializar(t banco.Transaccion) transaccion {
	var monto *string
	if t.Monto != nil {
		v := strconv.FormatInt(*t.Monto, 10)
		monto = &v
	}
	return transaccion{t.ID, t.SolicitudID, t.Fecha.UTC().Format("2006-01-02T15:04:05.999999999Z"), t.Tipo, t.Entrada, monto, t.Estado, t.Motivo, strconv.FormatInt(t.SaldoAnterior, 10), strconv.FormatInt(t.SaldoPosterior, 10)}
}
func respuesta(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func errorJSON(w http.ResponseWriter, code int, codigo, mensaje string) {
	respuesta(w, code, map[string]string{"codigo": codigo, "error": mensaje})
}
func metodo(w http.ResponseWriter, r *http.Request, esperado string) bool {
	if r.Method != esperado {
		w.Header().Set("Allow", esperado)
		errorJSON(w, 405, "metodo", "Método no permitido")
		return false
	}
	return true
}
func mismoOrigen(r *http.Request) bool {
	o, err := url.Parse(r.Header.Get("Origin"))
	if err != nil || o.Host == "" || o.User != nil || o.Path != "" || o.RawQuery != "" || o.Fragment != "" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return o.Scheme == scheme && strings.EqualFold(o.Host, r.Host) && r.Header.Get("Sec-Fetch-Site") != "cross-site"
}

func Handler(s banco.Operador) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/cuenta", func(w http.ResponseWriter, r *http.Request) {
		if !metodo(w, r, "GET") {
			return
		}
		e := s.Estado()
		respuesta(w, 200, map[string]string{"titular": e.Titular, "cuenta_id": e.CuentaID, "moneda": e.Moneda, "saldo_centimos": strconv.FormatInt(e.Saldo, 10)})
	})
	mux.HandleFunc("/api/transacciones", func(w http.ResponseWriter, r *http.Request) {
		if !metodo(w, r, "GET") {
			return
		}
		h := s.Estado().Historial
		lista := make([]transaccion, 0, len(h))
		for _, t := range h {
			lista = append(lista, serializar(t))
		}
		respuesta(w, 200, lista)
	})
	operar := func(tipo string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !metodo(w, r, "POST") {
				return
			}
			if !mismoOrigen(r) {
				errorJSON(w, 403, "origen", "Origen no autorizado")
				return
			}
			media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || media != "application/json" {
				errorJSON(w, 415, "contenido", "Se requiere JSON")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			d := json.NewDecoder(r.Body)
			d.DisallowUnknownFields()
			var q struct {
				ID    *string `json:"solicitud_id"`
				Monto *string `json:"monto"`
			}
			if err := d.Decode(&q); err != nil {
				var grande *http.MaxBytesError
				if errors.As(err, &grande) {
					errorJSON(w, 413, "tamano", "Solicitud demasiado grande")
				} else {
					errorJSON(w, 400, "json", "JSON inválido")
				}
				return
			}
			if err := d.Decode(&struct{}{}); err != io.EOF {
				var grande *http.MaxBytesError
				if errors.As(err, &grande) {
					errorJSON(w, 413, "tamano", "Solicitud demasiado grande")
				} else {
					errorJSON(w, 400, "json", "JSON inválido")
				}
				return
			}
			if q.ID == nil || q.Monto == nil || strings.TrimSpace(*q.ID) == "" || len(*q.ID) > 128 {
				errorJSON(w, 400, "solicitud", "Identificador y monto requeridos")
				return
			}
			t, err := s.Operar(banco.Solicitud{ID: *q.ID, Tipo: tipo, Monto: *q.Monto})
			if err != nil {
				switch {
				case errors.Is(err, banco.ErrConflicto):
					errorJSON(w, 409, "conflicto", "Identificador reutilizado con datos diferentes")
				case errors.Is(err, banco.ErrPersistencia):
					errorJSON(w, 503, "persistencia", "No se confirmó la persistencia; conserve el identificador")
				case errors.Is(err, banco.ErrIncierto):
					errorJSON(w, 503, "incierto", "Confirmación incierta; reinicie y reconcilie con el mismo identificador")
				default:
					errorJSON(w, 500, "tecnico", "Error técnico; conserve el identificador y reconcilie")
				}
				return
			}
			code := 200
			if t.Estado == "RECHAZADA" {
				code = 422
			}
			respuesta(w, code, serializar(t))
		}
	}
	mux.HandleFunc("/api/retiros", operar("RETIRO"))
	mux.HandleFunc("/api/abonos", operar("ABONO"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Método no permitido", 405)
			return
		}
		nombre := strings.TrimPrefix(r.URL.Path, "/")
		if nombre == "" {
			nombre = "index.html"
		}
		if nombre != "index.html" && nombre != "app.js" && nombre != "style.css" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		http.ServeFileFS(w, r, assets, nombre)
	})
	return mux
}
