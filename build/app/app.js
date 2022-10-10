import axios from "axios";
import express from "express";
import morgan from "morgan";

// Start up the Express app
const app = express();

// Log requests
app.use(morgan("dev"));

// Parse JSON body into req.body
app.use(express.json());

//
// GET endpoints
//
app.get(`/data`, async (req, res, next) => {
  try {
    const result = await axios({
      method: "GET",
      url: `http://${process.env.HOST_NAME}:${process.env.HOST_PORT}/products`,
    });

    res.status(200).send(result.data);
  } catch (err) {
    next(err);
  }
});

app.get(`/custom`, async (req, res, next) => {
  const url = req.query.url;
  if (url == undefined || url == "")
    return next({ message: "Specificare l'url da cercare", status: 400 });

  try {
    const result = await axios({
      method: "GET",
      url: `http://${url}`,
    });

    res.status(200).send(result.data);
  } catch (err) {
    next(err);
  }
});

// Handle errors
app.use((err, req, res, next) => {
  err.status = err.status || 500;
  console.log(err);
  res.status(err.status).json({
    error: err,
  });
});

export default app;
