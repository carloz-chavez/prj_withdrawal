# Documental: no hay ejecutor Gherkin configurado.
Característica: Consultar la cuenta mediante la web
  Escenario: Consultar el saldo exacto
    Dada una cuenta en PEN con saldo de 123 céntimos
    Cuando consulto la cuenta mediante HTTP
    Entonces recibo el saldo como la cadena "123"

  Escenario: Registrar un retiro y un abono
    Dada una cuenta con saldo suficiente
    Cuando registro un retiro y un abono con identificadores distintos
    Entonces ambos movimientos aparecen en el historial y actualizan el saldo

  Escenario: Rechazar fondos insuficientes o entrada inválida
    Dada una cuenta existente
    Cuando solicito un retiro mayor al saldo o un monto inválido
    Entonces el movimiento queda registrado como rechazado y el saldo no cambia

  Escenario: Reintentar una solicitud
    Dada una solicitud ya confirmada
    Cuando repito la misma solicitud con los mismos datos
    Entonces recibo la misma transacción sin duplicar el movimiento
    Pero si cambio el monto recibo un conflicto

  Escenario: Recuperar el estado
    Dada una operación confirmada en el almacenamiento
    Cuando se reinicia el servicio con la misma ruta
    Entonces se recuperan el saldo y el historial
