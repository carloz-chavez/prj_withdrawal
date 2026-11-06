
# Historias de Usuario y Reglas de Negocio

**Proyecto:** prj_withdrawal  
**Caso de Uso:** UC-001 - Retirar efectivo

---

## US-01: Retiro exitoso

**Escenario:** Successful withdrawal within balance.

**Historia de Usuario:**

As a: cliente autenticado

I want: retirar efectivo, indicando un monto

so that: disponer de mis ahorros y ver mi nuevo saldo

### Acceptance criteria

Dado que estoy autenticado, mi cuenta está activa y el ATM tiene efectivo, cuando ingreso un monto válido menor o igual a mi saldo, entonces el sistema descuenta el monto, registra la transacción como exitosa, dispensa el efectivo y muestra el nuevo saldo.

---

## US-02: Retiro rechazado por saldo insuficiente

**Escenario:** Declined withdrawal in excess of balance.

**Historia de Usuario:**

As a: cliente en sesión

I want: recibir un mensaje claro cuando mi saldo no alcanza

so that: elegir un monto menor o cancelar sin perder dinero

### Acceptance criteria

Cuando el monto solicitado es mayor que mi saldo, el sistema muestra "Saldo insuficiente. Intente con un monto menor", no descuenta nada y me permite ingresar otro monto o cancelar.

---

## US-03: Validación del monto

**Historia de Usuario:**

As a: cliente en sesión

I want: que el sistema valide el monto ingresado

so that: corregir errores antes de operar

### Acceptance criteria

Si el monto es negativo, cero, no es múltiplo de los billetes disponibles o excede el límite diario, el sistema muestra el error específico y vuelve a solicitar el monto.

---

## US-04: Integridad de la transacción

**Historia de Usuario:**

As a: banco (servicio ATM)

I want: que cada retiro sea atómico y quede registrado

so that: garantizar consistencia contable y auditoría

### Acceptance criteria

Una transacción se completa por entero o no se hace. Todo intento deja rastro en el historial (los fallidos según la política de auditoría).

---

## Reglas de Negocio Asociadas

- **RN-01:** No se permite realizar un retiro si el monto solicitado es mayor al saldo disponible.
- **RN-02:** No se permiten saldos negativos por retiros de efectivo (sin sobregiro autorizado).
- **RN-03:** Las transacciones deben ser atómicas (o se completa todo o no se hace nada).
