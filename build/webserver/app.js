import express from "express";
import morgan from "morgan";

import productRouter from "./routes/productRouter.js";

// Start up the Express app
const app = express();

// Log requests
app.use(morgan("dev"));

// Parse JSON body into req.body
app.use(express.json());

app.use("/products", productRouter);

// Handle errors
app.use((err, req, res, next) => {
  err.status = err.status || 500;
  console.log(err);
  res.status(err.status).json({
    error: err,
  });
});

export default app;
