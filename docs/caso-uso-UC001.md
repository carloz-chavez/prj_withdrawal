
# Caso de Uso: Realizar Retiro de Efectivo

**Versión:** 1.2.0

| Campo | Descripción |
|---|---|
| ID | UC-001 |
| Nombre | Realizar Retiro de Efectivo |
| Actor Principal | Cliente (Titular de la cuenta) |
| Actores Secundarios | Sistema Bancario (Core), Cajero Automático (ATM) / App Móvil (Canal) |
| Nivel | Objetivo de Usuario |

## 1. Descripción

Este caso de uso permite al Cliente solicitar una cantidad de efectivo de su cuenta bancaria.

El sistema valida si la cuenta tiene fondos suficientes y, en caso afirmativo, procesa la transacción descontando el monto.

Si los fondos son insuficientes, el sistema rechaza la solicitud y notifica al cliente sin realizar ningún movimiento financiero.

## 2. Precondiciones

- El Cliente está autenticado en el sistema (ATM, Banca Online o App).
- El Cliente tiene una cuenta activa con saldo disponible.
- El dispositivo (ATM) tiene efectivo disponible o la conexión a la red está operativa.

## 3. Flujo Principal (Escenario de Éxito)

Corresponde al escenario:

**Successful withdrawal within balance**

1. El Cliente selecciona la opción "Retirar efectivo" en el menú.
2. El Sistema solicita ingresar el monto a retirar.
3. El Cliente ingresa el monto (ej. 200.00) y confirma.
4. El Sistema verifica que el monto sea válido (mayor a 0 y múltiplo de billetes disponibles).
5. El Sistema consulta el saldo actual de la cuenta del Cliente.
6. El Sistema comprueba que el saldo sea mayor o igual al monto solicitado.
7. El Sistema descuenta el monto de la cuenta.
8. El Sistema registra la transacción como exitosa en el historial.
9. El Sistema dispensa el efectivo (si es ATM) o muestra el mensaje de éxito (si es digital).
10. El Sistema muestra el nuevo saldo disponible.
11. El Caso de Uso finaliza con éxito.

## 4. Flujos Alternativos

### 4.a. Saldo Insuficiente

Corresponde al escenario:

**Declined withdrawal in excess of balance**

En el paso 6 del flujo principal:

- El Sistema detecta que el saldo disponible es menor que el monto solicitado.
- El Sistema cancela la operación y NO descuenta ningún dinero.
- El Sistema muestra un mensaje de error: "Saldo insuficiente. Intente con un monto menor".
- El Sistema ofrece la opción de ingresar un nuevo monto o cancelar.
- Si el Cliente elige cancelar, el Caso de Uso termina. Si elige otro monto, vuelve al paso 3.

### 4.b. Monto Inválido o Límites Excedidos

En el paso 4 del flujo principal:

- El Sistema detecta que el monto es negativo, cero, no es múltiplo de billetes o excede el límite diario.
- El Sistema muestra un mensaje de validación indicando el error específico.
- El Sistema solicita al Cliente que ingrese un nuevo monto.
- Vuelve al paso 3.

## 5. Postcondiciones

**Si fue exitoso:**

El saldo de la cuenta se ha reducido en la cantidad solicitada y se ha generado un registro de transacción.

**Si fue rechazado:**

El saldo de la cuenta permanece inalterado y se ha registrado un intento fallido (opcional, según política de auditoría).

## 6. Reglas de Negocio Asociadas

- **RN-01:** No se permite realizar un retiro si el monto solicitado es mayor al saldo disponible.
- **RN-02:** No se permiten saldos negativos por retiros de efectivo (sin sobregiro autorizado).
- **RN-03:** Las transacciones deben ser atómicas (o se completa todo o no se hace nada).
