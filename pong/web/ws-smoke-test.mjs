import WebSocket from "ws";

const addr = process.argv[2];

async function newGame() {
  const resp = await fetch(`http://${addr}/pong/new_game`);
  return resp.json();
}

function connect(gameId) {
  return new WebSocket(`ws://${addr}/pong/ws/${gameId}`);
}

const gameId = await newGame();
console.log("created game", gameId);

const p1 = connect(gameId);
const p1Messages = [];
p1.on("message", (data) => p1Messages.push(JSON.parse(data.toString())));
await new Promise((resolve) => p1.on("open", resolve));

await new Promise((r) => setTimeout(r, 300));
console.log("player1 solo - last state:", p1Messages.filter((m) => m.type === "state").at(-1));

const p2 = connect(gameId);
const p2Messages = [];
p2.on("message", (data) => p2Messages.push(JSON.parse(data.toString())));
await new Promise((resolve) => p2.on("open", resolve));

await new Promise((r) => setTimeout(r, 100));
console.log("player2 joined message:", p2Messages.find((m) => m.type === "joined"));

p1.send(JSON.stringify({ type: "input", direction: "up" }));
await new Promise((r) => setTimeout(r, 300));
console.log("after p1 sends up - last state:", p1Messages.filter((m) => m.type === "state").at(-1));

p1.close();
p2.close();
process.exit(0);
