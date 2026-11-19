## Ejecutar la web

En Windows, con Go instalado, desde la raíz del proyecto:

```powershell
go run ./cmd/web -datos data/estado.json -titular "Cuenta de demostración" -saldo-inicial 100.00
```

Abrir **http://127.0.0.1:8080**. `-datos` configura la ruta del estado; `-escucha` cambia la dirección, cuyo valor predeterminado es `127.0.0.1:8080`. El titular y el saldo inicial solo se utilizan al crear una cuenta nueva. Para detener el servidor, usar Ctrl+C. El CLI persistente se ejecuta con `go run ./cmd/interactive -datos data/estado.json`; no abrir simultáneamente el CLI y la web con la misma ruta: el almacenamiento exige un único escritor.

La web muestra saldo, titular, identificador e historial; permite retiros y abonos y registra rechazos. Los montos se introducen como texto decimal en PEN y se procesan como céntimos enteros. Ante una respuesta incierta, conservar el identificador, reiniciar el servicio si es necesario, consultar el historial y reintentar el mismo intento; no iniciar otro para la misma operación. El almacenamiento JSON y el diario permiten recuperar el último estado confirmado. No borrar archivos corruptos o diarios incompletos para iniciar el servicio.

## API

| Método y ruta | Resultado |
| --- | --- |
| GET `/api/cuenta` | Titular, cuenta, moneda y saldo en céntimos como cadena. |
| GET `/api/transacciones` | Historial completo, con importes y saldos como cadenas de céntimos. |
| POST `/api/retiros` | Registra un retiro. |
| POST `/api/abonos` | Registra un abono. |

Los POST reciben JSON como `{"solicitud_id":"id-unico","monto":"12.50"}` y requieren un encabezado `Origin` correspondiente al servidor. Responden 200 si se confirmó, 422 si se registró un rechazo, 409 ante un conflicto de identificador y 503 ante un problema de persistencia o confirmación incierta. El historial incluye fecha UTC, entrada original, identificadores, estado, motivo y saldos. El servidor no expone el archivo de datos y no habilita CORS abierto.

## Pruebas y limitaciones

```powershell
go build ./...
go test -vet=off -count=1 ./... -v
go vet ./...
go test -race ./...
```




