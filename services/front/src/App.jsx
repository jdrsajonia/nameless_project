import { useEffect, useState } from "react";

// URLs inyectadas en build time por Vite (ver docker-compose: build.args).
const PAGOS_URL = import.meta.env.VITE_PAGOS_URL ?? "http://localhost:8000";
const MOTOR_URL = import.meta.env.VITE_MOTOR_URL ?? "http://localhost:8080";
const MOTOR_WS_URL = import.meta.env.VITE_MOTOR_WS_URL ?? "ws://localhost:8080";

function useHealth(url) {
  const [state, setState] = useState({ status: "cargando...", ok: false });
  useEffect(() => {
    let alive = true;
    fetch(`${url}/health`)
      .then((r) => r.json())
      .then((data) => alive && setState({ status: JSON.stringify(data), ok: true }))
      .catch((e) => alive && setState({ status: `error: ${e.message}`, ok: false }));
    return () => {
      alive = false;
    };
  }, [url]);
  return state;
}

export default function App() {
  const pagos = useHealth(PAGOS_URL);
  const motor = useHealth(MOTOR_URL);
  const [ws, setWs] = useState("conectando...");

  useEffect(() => {
    const socket = new WebSocket(`${MOTOR_WS_URL}/ws`);
    socket.onopen = () => setWs("WebSocket conectado");
    socket.onerror = () => setWs("WebSocket error");
    socket.onclose = () => setWs("WebSocket cerrado");
    return () => socket.close();
  }, []);

  const card = {
    border: "1px solid #ddd",
    borderRadius: 8,
    padding: "1rem 1.25rem",
    marginBottom: "1rem",
    fontFamily: "monospace",
  };

  return (
    <main
      style={{
        maxWidth: 720,
        margin: "3rem auto",
        padding: "0 1rem",
        fontFamily: "system-ui, sans-serif",
      }}
    >
      <h1>ArquiSoft — Prototipo 1</h1>
      <p>
        Esqueleto de infraestructura. Esta pantalla verifica que el
        front se comunica con los dos backends por REST y WebSocket.
      </p>

      <div style={card}>
        <strong>Pagos y usuarios</strong> {pagos.ok ? "✅" : "⏳"}
        <br />
        {pagos.status}
      </div>

      <div style={card}>
        <strong>Motor de apuestas</strong> {motor.ok ? "✅" : "⏳"}
        <br />
        {motor.status}
      </div>

      <div style={card}>
        <strong>WebSocket (motor)</strong>
        <br />
        {ws}
      </div>
    </main>
  );
}
