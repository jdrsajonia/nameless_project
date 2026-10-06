// Pruebas de la logica de HU-06. Se corren con:  node --test src/apuestas/
import test from "node:test";
import assert from "node:assert/strict";
import {
  apostar,
  crearMotorSimulado,
  formatoReloj,
  puntosPotenciales,
  segundosRestantes,
  validarMonto,
} from "./api.js";

test("puntos potenciales: 10 + monto x 0,5", () => {
  assert.equal(puntosPotenciales(100), 60);
  assert.equal(puntosPotenciales(1), 10.5);
});

test("cuenta regresiva", () => {
  const ahora = Date.parse("2026-10-06T12:00:00Z");
  assert.equal(segundosRestantes("2026-10-06T12:00:30Z", ahora), 30);
  assert.equal(segundosRestantes("2026-10-06T11:59:00Z", ahora), 0); // nunca negativa
  assert.equal(segundosRestantes("basura", ahora), 0);
  assert.equal(formatoReloj(75), "1:15");
  assert.equal(formatoReloj(5), "0:05");
});

test("validar monto", () => {
  assert.equal(validarMonto("").ok, false);
  assert.equal(validarMonto("").error, ""); // vacio: sin mensaje de error
  assert.equal(validarMonto("0").ok, false);
  assert.equal(validarMonto("-5").ok, false);
  assert.equal(validarMonto("12.5").ok, false);
  assert.equal(validarMonto("abc").ok, false);
  assert.deepEqual(validarMonto(" 40 "), { ok: true, monto: 40, error: "" });
  assert.equal(validarMonto("500", 100).error, "Saldo insuficiente");
  assert.equal(validarMonto("100", 100).ok, true);
});

const resp = (status, cuerpo) => async () => ({ ok: status < 300, status, json: async () => cuerpo });

test("apostar arma la peticion que espera el motor", async () => {
  let visto;
  const fetchFn = async (url, init) => {
    visto = { url, init };
    return { ok: true, status: 201, json: async () => ({ bet_id: "x" }) };
  };
  const r = await apostar({ mercadoId: "m1", opcion: "VER", monto: 100, clave: "k1", fetchFn, baseUrl: "http://motor" });
  assert.equal(r.ok, true);
  assert.equal(visto.url, "http://motor/apuestas");
  assert.equal(visto.init.method, "POST");
  assert.equal(visto.init.credentials, "include"); // manda la cookie de sesion
  assert.equal(visto.init.headers["Idempotency-Key"], "k1");
  assert.deepEqual(JSON.parse(visto.init.body), { market_id: "m1", option: "VER", amount: 100 });
});

test("apostar traduce los errores del motor", async () => {
  const base = { mercadoId: "m1", opcion: "VER", monto: 10, clave: "k" };
  let r = await apostar({ ...base, fetchFn: resp(422, { error: "Saldo insuficiente" }) });
  assert.deepEqual([r.ok, r.status, r.mensaje], [false, 422, "Saldo insuficiente"]);

  r = await apostar({ ...base, fetchFn: resp(409, { error: "Mercado cerrado" }) });
  assert.equal(r.mensaje, "Mercado cerrado");

  r = await apostar({ ...base, fetchFn: resp(401, { error: "no autorizado" }) });
  assert.match(r.mensaje, /sesion/i);

  r = await apostar({ ...base, fetchFn: async () => { throw new TypeError("fallo de red"); } });
  assert.equal(r.status, 0);
  assert.match(r.mensaje, /conectar/i);

  r = await apostar({ ...base, fetchFn: async () => ({ ok: false, status: 500, json: async () => { throw new Error("no json"); } }) });
  assert.equal(r.ok, false);
});

test("motor simulado descuenta el saldo y rechaza lo que no alcanza", async () => {
  const motor = crearMotorSimulado({ saldoInicial: 100 });
  const base = { mercadoId: "m1", opcion: "VER", clave: "k", fetchFn: motor.fetchFn };

  const ok = await apostar({ ...base, monto: 60 });
  assert.equal(ok.ok, true);
  assert.equal(ok.apuesta.potential_points, 40);
  assert.equal(motor.saldo(), 40);

  const mal = await apostar({ ...base, monto: 60 });
  assert.equal(mal.mensaje, "Saldo insuficiente");
  assert.equal(motor.saldo(), 40);
});
