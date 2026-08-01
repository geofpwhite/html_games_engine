import { createClient } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { PongService, Direction } from "./src/pong/v1/pong_pb.js";

const addr = process.argv[2];
const transport = createGrpcTransport({ baseUrl: `http://${addr}`, httpVersion: "2" });
const client = createClient(PongService, transport);

async function* requests() {
  yield { payload: { case: "join", value: { gameId: "" } } };
  await new Promise((r) => setTimeout(r, 50));
  yield { payload: { case: "input", value: { direction: Direction.UP } } };
  await new Promise((r) => setTimeout(r, 500));
}

let count = 0;
for await (const msg of client.play(requests())) {
  console.log(JSON.stringify(msg));
  count++;
  if (count > 8) break;
}
process.exit(0);
