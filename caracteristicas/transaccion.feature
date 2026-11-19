# language: es
Característica: Integridad de retiros y abonos con registros JSON
  Como responsable del servicio bancario
  Quiero confirmar saldo e historial juntos en PEN
  Para recuperar operaciones sin movimientos parciales ni duplicados

  Antecedentes:
    Dado un almacenamiento disponible
    Y una cuenta con saldo confirmado de "100.00" PEN

  Escenario: Confirmar un retiro con su registro
    Cuando solicito retirar "30.00"
    Entonces la operación queda confirmada
    Y el saldo confirmado es "70.00"
    Y existe un único registro exitoso de retiro
    Y el registro contiene identificador, fecha UTC y saldos anterior y posterior

  Escenario: Confirmar un abono con su registro
    Cuando solicito abonar "25.00"
    Entonces la operación queda confirmada
    Y el saldo confirmado es "125.00"
    Y existe un único registro exitoso de abono

  Escenario: Registrar un retiro rechazado por fondos insuficientes
    Cuando solicito retirar "120.00"
    Entonces la solicitud queda rechazada por fondos insuficientes
    Y el saldo confirmado permanece en "100.00"
    Y existe un registro rechazado con saldos anterior y posterior iguales

  Esquema del escenario: Rechazar montos inválidos
    Cuando solicito una operación de "<tipo>" por "<entrada>"
    Entonces la solicitud queda rechazada por monto inválido
    Y el saldo confirmado permanece en "100.00"
    Y existe un registro rechazado con el motivo correspondiente

    Ejemplos:
      | tipo   | entrada              |
      | RETIRO | 0                    |
      | RETIRO | -1.00                |
      | ABONO  | 0                    |
      | ABONO  | -1.00                |
      | RETIRO | NaN                  |
      | ABONO  | +Inf                 |
      | RETIRO | abc                  |
      | ABONO  | 1.001                |
      | ABONO  | 92233720368547758.08 |

  Escenario: Fallar antes de confirmar el almacenamiento
    Dado que la escritura del candidato falla antes de la confirmación
    Cuando solicito retirar "30.00"
    Entonces recibo un error de persistencia
    Y no se anuncia éxito
    Y el estado confirmado conserva su saldo e historial anteriores

  Escenario: No poder persistir un rechazo
    Dado que el almacenamiento deja de estar disponible
    Cuando solicito retirar "120.00"
    Entonces recibo un error de auditoría
    Y el saldo permanece en "100.00"
    Y no se afirma que el intento quedó registrado

  Escenario: Recuperar una operación confirmada al reiniciar
    Dado un abono de "25.00" confirmado
    Cuando reinicio el servicio
    Entonces el saldo recuperado es "125.00"
    Y el historial contiene el abono una sola vez

  Escenario: Interrumpir una confirmación
    Dado que el proceso se interrumpe durante la confirmación de un retiro
    Cuando reinicio el servicio
    Entonces recupero un estado completo y coherente
    Y nunca recupero un saldo modificado sin su registro correspondiente
    Y si la recuperación es ambigua se bloquean nuevas operaciones

  Escenario: Reintentar una solicitud ya confirmada
    Dado un retiro de "30.00" confirmado con la solicitud "solicitud-1"
    Cuando repito el retiro con la solicitud "solicitud-1"
    Entonces recupero el resultado original
    Y el saldo confirmado es "70.00"
    Y existe un único registro para esa solicitud

  Escenario: Detectar almacenamiento corrupto
    Dado un archivo persistente corrupto o con invariantes incumplidos
    Cuando inicio el servicio
    Entonces recibo un error de recuperación
    Y no se crea una cuenta nueva sobre el estado existente

  Escenario: Serializar retiros concurrentes
    Cuando dos solicitudes distintas intentan retirar "80.00" simultáneamente
    Entonces exactamente una queda confirmada
    Y la otra queda rechazada por fondos insuficientes
    Y el saldo confirmado es "20.00"
    Y ambos resultados quedan registrados

  Escenario: Impedir un segundo escritor
    Dado otro proceso con acceso exclusivo al almacenamiento
    Cuando intento iniciar un segundo servicio escritor
    Entonces recibo un error de almacenamiento en uso
    Y no se modifica el estado persistente

  Escenario: Bloquear un diario incompleto
    Dado que el diario termina en un registro incompleto
    Cuando inicio el servicio
    Entonces recibo un error de confirmación incierta
    Y no se descarta automáticamente el registro incompleto
    Y no se permiten nuevas operaciones

  Escenario: Consultar historial después de reiniciar el CLI
    Dado un retiro de "30.00" y un abono de "25.00" confirmados
    Cuando reinicio el CLI y consulto el historial
    Entonces veo ambos registros con identificador, fecha, monto, estado y saldo resultante
    Y el saldo confirmado es "95.00"
