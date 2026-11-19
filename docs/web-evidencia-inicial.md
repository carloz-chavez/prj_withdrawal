# Punto 4: arquitectura y verificación web

Línea base: `go test -vet=off -count=1 ./...` pasó para los paquetes existentes antes de crear `internal/web` (Go 1.27.3, Windows/amd64).

Historia inicial: consultar titular, identificador, moneda y saldo exacto de la cuenta. El saldo se devuelve como cadena de céntimos para evitar pérdida de precisión en JavaScript.

RED: `go test -vet=off ./internal/web -run TestConsultarCuenta -v` ejecutó la prueba y falló con `status: got 404, want 200` usando un handler provisional.

GREEN: el mismo comando pasó tras añadir `GET /api/cuenta`. Para ejecutar esta prueba se utilizó una caché temporal de Go, debido a un error de acceso en la caché predeterminada.

Para el siguiente incremento, las pruebas `TestOperacionesHTTP` y `TestErroresHTTP` fallaron inicialmente con HTTP 404 para las rutas de operación. Después de implementar las rutas, ambas pasaron. Los escenarios de `caracteristicas/web.feature` son documentales; el proyecto no configura un ejecutor Gherkin.

La aplicación usa `net/http` y recursos embebidos, un único servicio `internal/banco` y un único almacenamiento `internal/persistencia` por proceso. El servicio controla los importes enteros en céntimos, serializa las operaciones y guarda éxitos y rechazos. Las respuestas HTTP representan los importes monetarios como cadenas. El navegador conserva el identificador de cada intento hasta recibir una respuesta concluyente o reconciliarlo con el historial. Un reintento idéntico no duplica la operación; reutilizar el ID con otros datos produce conflicto.

El almacenamiento utiliza JSON versionado y un diario de estados con SHA-256. La recuperación verifica las invariantes y el diario; una confirmación incierta requiere reiniciar y reconciliar. El bloqueo exclusivo impide abrir simultáneamente el mismo estado desde el CLI y la web. El adaptador de persistencia está implementado para Windows; los demás sistemas no cuentan con ese soporte.

Verificación del 19 de noviembre de 2026 con Go 1.27.3 en Windows/amd64: `go build ./...` y `go test -vet=off -count=1 ./... -v` pasaron. Hay 39 funciones de prueba principales, incluidas siete de `internal/web`, además de subpruebas. Mediante HTTP sobre datos sintéticos se verificaron consulta, retiro, abono, repetición idempotente, rechazos y recuperación del saldo e historial tras reiniciar. `go vet ./...` informó las seis advertencias existentes de `headers/test_bdd.go`. `go test -race ./...` no pudo compilar porque el compilador C disponible no admite modo de 64 bits y también mostró esas advertencias de vet.

La verificación visual de escritorio y móvil, teclado, foco y estados de interfaz queda pendiente: el navegador controlado no estuvo disponible y Playwright CLI no estaba disponible sin instalación. Las pruebas HTTP no sustituyen esa comprobación visual. La aplicación es una demostración local académica, no un sistema bancario de producción.
