// Proves the Kafka per-node fan-out actually works: registers 2 users,
// connects each to a DIFFERENT backend node's WebSocket (backend-1 / backend-2
// from backend/docker-compose.yml), then sends a chat message over Alice's
// own WebSocket. Delivery only works if backend-1 published it to Kafka and
// backend-2 (which holds Bob's connection) picked it up and forwarded it.
//
// Sending via the REST endpoint (POST /chat/sendmessage) would NOT trigger
// this — that endpoint only persists to MongoDB (see chat_controller.go).
// Kafka publish only happens on the WebSocket path (see chat_ws.go's
// handleIncomingWebsocketMessages -> SendMessageWithRetry).
//
// Usage: node backend/scripts/demo-cross-node-chat.mjs
// Requires: docker compose -f backend/docker-compose.yml up -d --build

const ts = Date.now();
const userA = { email: `a-${ts}@demo.local`, password: "password123", firstName: "Alice", lastName: "Demo" };
const userB = { email: `b-${ts}@demo.local`, password: "password123", firstName: "Bob", lastName: "Demo" };

async function signup(base, u) {
  const res = await fetch(`${base}/user/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(u),
  });
  const body = await res.json();
  if (!res.ok) throw new Error(`signup ${u.email} failed: ${res.status} ${JSON.stringify(body)}`);
  return body;
}

async function login(base, u) {
  const res = await fetch(`${base}/user/signin`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: u.email, password: u.password }),
  });
  const body = await res.json();
  if (!res.ok) throw new Error(`login ${u.email} failed: ${res.status} ${JSON.stringify(body)}`);
  return { token: body.token, id: body.result._id };
}

async function main() {
  console.log("Registering users on backend-1 (5001)...");
  await signup("http://localhost:5001", userA);
  await signup("http://localhost:5001", userB);

  const a = await login("http://localhost:5001", userA);
  const b = await login("http://localhost:5002", userB);
  console.log("Alice id:", a.id, "-> connecting WS to backend-1 (5001)");
  console.log("Bob   id:", b.id, "-> connecting WS to backend-2 (5002)");

  const wsA = new WebSocket(`ws://localhost:5001/chat/ws?token=${a.token}`);
  const wsB = new WebSocket(`ws://localhost:5002/chat/ws?token=${b.token}`);

  const received = new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("Timed out waiting for message on Bob's WS (backend-2)")), 15000);
    wsB.addEventListener("message", (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.content) {
        clearTimeout(timeout);
        resolve(msg);
      }
    });
    wsB.addEventListener("error", (e) => reject(e));
  });

  await Promise.all([
    new Promise((resolve, reject) => { wsA.addEventListener("open", resolve); wsA.addEventListener("error", reject); }),
    new Promise((resolve, reject) => { wsB.addEventListener("open", resolve); wsB.addEventListener("error", reject); }),
  ]);
  console.log("Alice's WebSocket connected to backend-1, Bob's WebSocket connected to backend-2.");

  await new Promise((r) => setTimeout(r, 1000));

  console.log("Alice sending message over HER OWN WebSocket (backend-1) -> Kafka -> should reach Bob on backend-2...");
  wsA.send(JSON.stringify({ sender: a.id, receiver: b.id, content: "Hello from Alice via node backend-1!" }));

  const msg = await received;
  console.log("\n=== SUCCESS: Bob received on backend-2's WebSocket ===");
  console.log(msg);
  wsA.close();
  wsB.close();
  process.exit(0);
}

main().catch((err) => {
  console.error("DEMO FAILED:", err.message || err);
  process.exit(1);
});
