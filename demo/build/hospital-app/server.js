import dotenv from "dotenv";
import http from "http";

import app from "./app.js";

dotenv.config({ path: "./config.env" });

process.on("uncaughtException", (err) => {
  console.log("[app] Uncaught exception. Shutting down.");
  console.log(err.name, err.message, err);
  process.exit(1);
});

const server = http.createServer(app);
const port = process.env.PORT || 80;

server.listen(port, () => {
  console.log(`[app] Http web server listening on port ${port}`);
});

process.on("unhandledRejection", (err) => {
  console.log("[app] Unhandled rejection. Shutting down.");
  console.log(err.name, err.message);
  server.close(() => {
    process.exit(1);
  });
});
