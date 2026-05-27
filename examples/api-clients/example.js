#!/usr/bin/env node
// v-star API example — present value, simulation, annuity, reserve.
// Requires Node.js 18+ (fetch is built-in) or Node 16 with node-fetch.
//
// Usage: node example.js
// Requires v-star server running on http://localhost:8080.

const BASE = "http://localhost:8080";

async function post(path, data) {
  const resp = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!resp.ok) throw new Error(`${resp.status}: ${await resp.text()}`);
  return resp.json();
}

async function main() {
  // --- Present value ---
  const pv = await post("/value", {
    interest_rate: 0.05,
    records: [{ sum_assured: 100000, term: 20 }],
  });
  console.log(`PV: ${pv.total_pv.toFixed(2)} (${pv.record_count} policies)`);

  // --- Monte Carlo simulation + risk metrics ---
  const mc = await post("/simulate", {
    num_paths: 10000,
    steps: 10,
    initial_rate: 0.05,
    drift: 0.02,
    volatility: 0.15,
  });
  console.log(`MC: VaR95=${mc.var_95.toFixed(4)}, CTE95=${mc.cte_95.toFixed(4)}`);

  // --- Annuity / NSP ---
  const ann = await post("/annuity", {
    interest_rate: 0.05,
    qxs: Array(111).fill(0.001),
    age: 30,
    amount: 1000,
    computation: "whole_life_immediate",
  });
  console.log(`Ann: PV=${ann.present_value.toFixed(2)}`);

  // --- Reserve ---
  const res = await post("/reserve", {
    interest_rate: 0.05,
    qxs: Array(111).fill(0.001),
    age: 30,
    term: 20,
    sum_assured: 100000,
    method: "net_premium",
  });
  console.log(`Res: reserve=${res.reserve.toFixed(2)}`);
}

main().catch(console.error);
