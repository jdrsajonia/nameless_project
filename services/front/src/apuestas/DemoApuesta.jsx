import { useMemo, useState } from "react";
import FormularioApuesta from "./FormularioApuesta.jsx";
import { crearMotorSimulado } from "./api.js";

// Demo de HU-06 con un mercado y un motor de mentira, para ver la pantalla
// sin backend ni sesion. Se borra cuando el mercado real (HU-11) y el login
// (HU-02) existan: ahi se usa <FormularioApuesta mercado={...} /> directo.

function mercadoDemo() {
  return {
    id: crypto.randomUUID(),
    pregunta: "¿Quién será el primero al terminar el minuto?",
    opciones: ["VER", "NOR", "HAM", "LEC"],
    cierra_en: new Date(Date.now() + 30_000).toISOString(),
  };
}

export default function DemoApuesta() {
  const [mercado, setMercado] = useState(mercadoDemo);
  const motor = useMemo(() => crearMotorSimulado({ saldoInicial: 1000 }), []);
  const [saldo, setSaldo] = useState(motor.saldo());

  return (
    <section>
      <h2>HU-06 · Apostar (demo con datos simulados)</h2>
      <FormularioApuesta
        mercado={mercado}
        saldo={saldo}
        fetchFn={motor.fetchFn}
        onAceptada={() => setSaldo(motor.saldo())}
      />
      <button type="button" onClick={() => setMercado(mercadoDemo())}>
        Abrir otro mercado de prueba (30 s)
      </button>
    </section>
  );
}
