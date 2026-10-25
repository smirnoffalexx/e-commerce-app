import { createServer } from "node:http";
import next from "next";

const port = Number(process.env.PORT ?? 3000);
const hostname = process.env.HOSTNAME ?? "0.0.0.0";
const trustProxy = process.env.TRUST_PROXY === "true";
const forwardedHeaders = ["x-forwarded-for", "x-forwarded-host", "x-forwarded-proto", "x-forwarded-port", "x-real-ip"];

const app = next({ dev: false, hostname, port });
const handle = app.getRequestHandler();
await app.prepare();

const server = createServer((req, res) => {
  if (!trustProxy) {
    for (const name of forwardedHeaders) {
      delete req.headers[name];
    }
    if (req.socket.remoteAddress) {
      req.headers["x-forwarded-for"] = req.socket.remoteAddress;
    }
  }
  handle(req, res);
});

server.listen(port, hostname, () => {
  console.log(`e-commerce-app frontend ready on http://${hostname}:${port}`);
});

for (const signal of ["SIGTERM", "SIGINT"]) {
  process.on(signal, () => {
    server.close(() => process.exit(0));
    setTimeout(() => process.exit(1), 10_000).unref();
  });
}
