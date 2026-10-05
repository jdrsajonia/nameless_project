import { useEffect, useState } from "react";
import { apostar, formatoReloj, puntosPotenciales, segundosRestantes, validarMonto } from "./api.js";

// Formulario de HU-06: elegir opcion y monto en el mercado abierto.
//
// Props:
//   mercado   { id, pregunta, opciones: string[], cierra_en: ISO }
//   saldo     (opcional) saldo en tokens a mostrar; si viene, valida el monto
//   fetchFn   (opcional) reemplaza a fetch, p. ej. el motor simulado
//   onAceptada(apuesta) (opcional) se llama cuando el motor acepta la apuesta
//
// La cuenta regresiva es solo informativa: aunque llegue a 0 despues que el
// servidor, el motor rechaza la apuesta con "Mercado cerrado" (RNF-05).

const estilos = {
  tarjeta: { border: "1px solid #ddd", borderRadius: 8, padding: "1rem 1.25rem", marginBottom: "1rem" },
  fila: { display: "flex", gap: 8, flexWrap: "wrap", margin: "0.75rem 0" },
  opcion: (activa) => ({
    padding: "0.5rem 1rem",
    borderRadius: 6,
    border: activa ? "2px solid #2563eb" : "1px solid #bbb",
    background: activa ? "#eff6ff" : "#fff",
    cursor: "pointer",
    font: "inherit",
  }),
  input: { padding: "0.5rem", font: "inherit", width: 140, borderRadius: 6, border: "1px solid #bbb" },
  boton: (deshabilitado) => ({
    padding: "0.6rem 1.25rem",
    font: "inherit",
    borderRadius: 6,
    border: "none",
    color: "#fff",
    background: deshabilitado ? "#9ca3af" : "#2563eb",
    cursor: deshabilitado ? "not-allowed" : "pointer",
  }),
  error: { color: "#b91c1c", margin: "0.5rem 0 0" },
  ok: { color: "#15803d", margin: "0.5rem 0 0" },
  nota: { color: "#6b7280", fontSize: "0.85rem" },
};

export default function FormularioApuesta({ mercado, saldo = null, fetchFn, onAceptada }) {
  const [opcion, setOpcion] = useState(null);
  const [texto, setTexto] = useState("");
  const [segundos, setSegundos] = useState(() => segundosRestantes(mercado.cierra_en));
  const [enviando, setEnviando] = useState(false);
  const [resultado, setResultado] = useState(null); // { ok, mensaje }

  // Reloj: se reinicia si cambia el mercado.
  useEffect(() => {
    setSegundos(segundosRestantes(mercado.cierra_en));
    const id = setInterval(() => setSegundos(segundosRestantes(mercado.cierra_en)), 250);
    return () => clearInterval(id);
  }, [mercado.cierra_en, mercado.id]);

  useEffect(() => {
    setOpcion(null);
    setTexto("");
    setResultado(null);
  }, [mercado.id]);

  const cerrado = segundos === 0;
  const monto = validarMonto(texto, saldo);
  const puedeEnviar = !cerrado && !enviando && opcion !== null && monto.ok;

  async function enviar(e) {
    e.preventDefault();
    if (!puedeEnviar) return;
    setEnviando(true);
    setResultado(null);
    // Una clave nueva por clic: el boton queda bloqueado mientras se envia, asi
    // un doble clic no manda dos apuestas.
    const r = await apostar({
      mercadoId: mercado.id,
      opcion,
      monto: monto.monto,
      clave: crypto.randomUUID(),
      fetchFn,
    });
    setEnviando(false);

    if (r.ok) {
      setResultado({
        ok: true,
        mensaje: `Apuesta registrada: ${monto.monto} tokens a ${opcion}. Si aciertas sumas ${r.apuesta.potential_points} puntos.`,
      });
      setTexto("");
      onAceptada?.(r.apuesta);
    } else {
      setResultado({ ok: false, mensaje: r.mensaje });
    }
  }

  return (
    <form onSubmit={enviar} style={estilos.tarjeta} aria-label="Apostar en el mercado abierto">
      <strong>{mercado.pregunta}</strong>
      <div aria-live="polite" style={{ margin: "0.25rem 0", fontFamily: "monospace" }}>
        {cerrado ? "Mercado cerrado" : `Cierra en ${formatoReloj(segundos)}`}
      </div>

      <div style={estilos.fila} role="radiogroup" aria-label="Opciones">
        {mercado.opciones.map((o) => (
          <button
            type="button"
            key={o}
            role="radio"
            aria-checked={opcion === o}
            disabled={cerrado}
            onClick={() => setOpcion(o)}
            style={estilos.opcion(opcion === o)}
          >
            {o}
          </button>
        ))}
      </div>

      <label>
        Monto en tokens{" "}
        <input
          style={estilos.input}
          inputMode="numeric"
          value={texto}
          disabled={cerrado}
          onChange={(e) => setTexto(e.target.value)}
          placeholder="Ej: 100"
        />
      </label>
      {saldo !== null && <span style={{ marginLeft: 12 }}>Saldo: {saldo} tokens</span>}

      <p style={estilos.nota}>
        {monto.ok
          ? `Si aciertas sumas ${puntosPotenciales(monto.monto)} puntos (10 + monto x 0,5).`
          : "Elige una opcion e ingresa un monto."}
      </p>
      {monto.error && <p style={estilos.error}>{monto.error}</p>}

      <button type="submit" disabled={!puedeEnviar} style={estilos.boton(!puedeEnviar)}>
        {enviando ? "Enviando..." : "Apostar"}
      </button>

      <div aria-live="polite">
        {resultado && <p style={resultado.ok ? estilos.ok : estilos.error}>{resultado.mensaje}</p>}
      </div>
      <p style={estilos.nota}>El cierre lo decide el servidor, no esta cuenta regresiva.</p>
    </form>
  );
}
