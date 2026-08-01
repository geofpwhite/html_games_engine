import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PongService, Direction } from "./pong/v1/pong_pb.js";

const BACKEND_URL = "https://geof.geofpwhite.us";
const FIELD = 100; // matches fieldWidth/fieldHeight in pong/pong.go

const canvas = document.getElementById("pong-canvas");
const ctx2d = canvas.getContext("2d");
const statusEl = document.getElementById("pong-status");
const gameID = canvas.dataset.gameId || "";

// The outgoing half of the bidi stream. connect-web pulls from this as an
// AsyncIterable; push() feeds it from the join call below and from keyboard
// events, waking the iterator if it's currently waiting on an empty queue.
function createOutbox() {
	const queue = [];
	let wake = null;
	return {
		push(msg) {
			queue.push(msg);
			if (wake) {
				const resolve = wake;
				wake = null;
				resolve();
			}
		},
		async *[Symbol.asyncIterator]() {
			for (;;) {
				while (queue.length === 0) {
					await new Promise((resolve) => { wake = resolve; });
				}
				yield queue.shift();
			}
		},
	};
}

const outbox = createOutbox();
outbox.push({ payload: { case: "join", value: { gameId: gameID } } });

let currentDirection = Direction.STOP;
function sendDirection(dir) {
	if (dir === currentDirection) return;
	currentDirection = dir;
	outbox.push({ payload: { case: "input", value: { direction: dir } } });
}

window.addEventListener("keydown", (e) => {
	if (e.key === "ArrowUp" || e.key === "w" || e.key === "W") sendDirection(Direction.UP);
	if (e.key === "ArrowDown" || e.key === "s" || e.key === "S") sendDirection(Direction.DOWN);
});
window.addEventListener("keyup", (e) => {
	const releasingUp = (e.key === "ArrowUp" || e.key === "w" || e.key === "W") && currentDirection === Direction.UP;
	const releasingDown = (e.key === "ArrowDown" || e.key === "s" || e.key === "S") && currentDirection === Direction.DOWN;
	if (releasingUp || releasingDown) sendDirection(Direction.STOP);
});

function draw(state) {
	const w = canvas.width;
	const h = canvas.height;
	const sx = w / FIELD;
	const sy = h / FIELD;

	ctx2d.fillStyle = "#111";
	ctx2d.fillRect(0, 0, w, h);

	ctx2d.fillStyle = "#eee";
	ctx2d.font = "20px monospace";
	ctx2d.textAlign = "center";
	ctx2d.fillText(`${state.score1} : ${state.score2}`, w / 2, 30);

	const paddleH = 20 * sy;
	const paddleW = 2 * sx;
	ctx2d.fillRect(0, state.paddle1Y * sy - paddleH / 2, paddleW, paddleH);
	ctx2d.fillRect(w - paddleW, state.paddle2Y * sy - paddleH / 2, paddleW, paddleH);

	ctx2d.beginPath();
	ctx2d.arc(state.ballX * sx, state.ballY * sy, 4, 0, Math.PI * 2);
	ctx2d.fill();
}

async function main() {
	const transport = createConnectTransport({ baseUrl: BACKEND_URL });
	const client = createClient(PongService, transport);

	try {
		for await (const msg of client.play(outbox)) {
			switch (msg.payload.case) {
				case "joined": {
					const { playerIndex } = msg.payload.value;
					statusEl.textContent = `You are player ${playerIndex + 1}. Share this page's URL to invite an opponent.`;
					break;
				}
				case "state":
					draw(msg.payload.value);
					break;
				case "error":
					statusEl.textContent = `Error: ${msg.payload.value.message}`;
					break;
				default:
					break;
			}
		}
		statusEl.textContent = "Disconnected.";
	} catch (err) {
		statusEl.textContent = `Disconnected: ${err}`;
	}
}

main();
