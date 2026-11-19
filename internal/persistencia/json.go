package persistencia

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"prj_withdrawal/internal/banco"
)

type registro struct {
	Estado banco.Estado `json:"estado"`
	SHA256 string       `json:"sha256"`
}
type archivo struct {
	mu                sync.Mutex
	ruta              string
	lock              *os.File
	cerrado, incierto bool
	// Punto de inyección interno para ensayar fallos del reemplazo real.
	reemplazo func(string, string) error
}

func Abrir(ruta string) (Archivo, error) {
	absoluta, err := filepath.Abs(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", banco.ErrPersistencia, err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluta), 0700); err != nil {
		return nil, fmt.Errorf("%w: %v", banco.ErrPersistencia, err)
	}
	if info, err := os.Stat(absoluta); err == nil && info.IsDir() {
		return nil, banco.ErrPersistencia
	}
	l, err := bloquear(absoluta + ".lock")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", banco.ErrEnUso, err)
	}
	a := &archivo{ruta: absoluta, lock: l, reemplazo: reemplazar}
	if _, err := a.recuperar(); err != nil && !errors.Is(err, os.ErrNotExist) {
		l.Close()
		return nil, err
	}
	return a, nil
}

func (a *archivo) Cerrar() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrado {
		return nil
	}
	a.cerrado = true
	return a.lock.Close()
}
func (a *archivo) Cargar() (banco.Estado, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrado {
		return banco.Estado{}, os.ErrClosed
	}
	if a.incierto {
		return banco.Estado{}, banco.ErrIncierto
	}
	return a.recuperar()
}

func codificar(e banco.Estado) ([]byte, error) { return json.Marshal(e) }
func suma(e banco.Estado) string {
	b, _ := codificar(e)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func decodificar(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return banco.ErrCorrupto
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return banco.ErrCorrupto
	}
	return nil
}
func extension(prev, next banco.Estado) bool {
	if next.CuentaID != prev.CuentaID || next.Titular != prev.Titular || next.Moneda != prev.Moneda || next.SaldoInicial != prev.SaldoInicial || len(next.Historial) != len(prev.Historial)+1 {
		return false
	}
	for i := range prev.Historial {
		if !reflect.DeepEqual(prev.Historial[i], next.Historial[i]) {
			return false
		}
	}
	return true
}

func (a *archivo) recuperar() (banco.Estado, error) {
	var vacio banco.Estado
	datos, err := os.ReadFile(a.ruta + ".journal")
	var estados []banco.Estado
	if err != nil && !os.IsNotExist(err) {
		return vacio, err
	}
	if len(datos) > 0 {
		if datos[len(datos)-1] != '\n' {
			return vacio, banco.ErrIncierto
		}
		for _, linea := range bytes.Split(datos[:len(datos)-1], []byte{'\n'}) {
			var r registro
			if decodificar(linea, &r) != nil || r.SHA256 != suma(r.Estado) || banco.ValidarEstado(r.Estado) != nil {
				return vacio, banco.ErrCorrupto
			}
			if len(estados) > 0 && !extension(estados[len(estados)-1], r.Estado) {
				return vacio, banco.ErrCorrupto
			}
			estados = append(estados, r.Estado)
		}
	}
	b, err := os.ReadFile(a.ruta)
	if err != nil && !os.IsNotExist(err) {
		return vacio, err
	}
	if len(estados) == 0 {
		if err != nil {
			return vacio, err
		}
		if decodificar(b, &vacio) != nil || banco.ValidarEstado(vacio) != nil {
			return banco.Estado{}, banco.ErrCorrupto
		}
		return vacio, nil
	}
	ultimo := estados[len(estados)-1]
	if err == nil {
		var visible banco.Estado
		if decodificar(b, &visible) != nil || banco.ValidarEstado(visible) != nil {
			return vacio, banco.ErrCorrupto
		}
		reconocido := false
		for _, e := range estados {
			if reflect.DeepEqual(e, visible) {
				reconocido = true
				break
			}
		}
		if !reconocido {
			return vacio, banco.ErrIncierto
		}
		if reflect.DeepEqual(visible, ultimo) {
			return ultimo, nil
		}
	}
	if err := a.proyectar(ultimo); err != nil {
		return vacio, fmt.Errorf("%w: %v", banco.ErrIncierto, err)
	}
	return ultimo, nil
}

func (a *archivo) Confirmar(e banco.Estado) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrado {
		return banco.ErrPersistencia
	}
	if a.incierto {
		return banco.ErrIncierto
	}
	if banco.ValidarEstado(e) != nil {
		return banco.ErrCorrupto
	}
	previo, err := a.recuperar()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !extension(previo, e) {
		return banco.ErrConflicto
	}
	// Si existe un documento importado sin diario, conservar primero su base.
	if err == nil {
		info, statErr := os.Stat(a.ruta + ".journal")
		if os.IsNotExist(statErr) || (statErr == nil && info.Size() == 0) {
			if err := a.anexar(previo); err != nil {
				return err
			}
		}
	}
	if err := a.anexar(e); err != nil {
		return err
	}
	if err := a.proyectar(e); err != nil {
		a.incierto = true
		return fmt.Errorf("%w: %v", banco.ErrIncierto, err)
	}
	return nil
}
func (a *archivo) anexar(e banco.Estado) error {
	b, err := json.Marshal(registro{e, suma(e)})
	if err != nil {
		return fmt.Errorf("%w: %v", banco.ErrPersistencia, err)
	}
	b = append(b, '\n')
	f, err := os.OpenFile(a.ruta+".journal", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", banco.ErrPersistencia, err)
	}
	n, werr := f.Write(b)
	serr := f.Sync()
	cerr := f.Close()
	if werr != nil || n != len(b) || serr != nil || cerr != nil {
		a.incierto = true
		return fmt.Errorf("%w: escritura=%v sync=%v cierre=%v", banco.ErrIncierto, werr, serr, cerr)
	}
	return nil
}
func (a *archivo) proyectar(e banco.Estado) error {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(a.ruta), ".us04-*.tmp")
	if err != nil {
		return err
	}
	nombre := f.Name()
	defer os.Remove(nombre)
	n, err := f.Write(b)
	if err != nil || n != len(b) {
		f.Close()
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return a.reemplazo(nombre, a.ruta)
}
