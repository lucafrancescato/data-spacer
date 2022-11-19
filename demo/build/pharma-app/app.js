import axios from "axios";
import express from "express";
import morgan from "morgan";

const MIN = 1;
const MAX = 101;

// Start up the Express app
const app = express();

// Log requests
app.use(morgan("dev"));

// Parse JSON body into req.body
app.use(express.json());

//
// GET endpoints
//
app.get(`/results`, async (req, res, next) => {
  try {
    const result = await axios({
      method: "GET",
      url: `http://${process.env.HOST_NAME}:${process.env.HOST_PORT}/patients`,
    });

    let sumAge = 0;
    let sumExams = 0;
    let sumChildren = 0;
    let n = 0;
    for (const el of result.data.res.data) {
      if (el.risk == "LOW") continue;
      sumAge += el.age;
      sumExams += el.medicalExams;
      sumChildren += el.children;
      n++;
    }

    const processedData = {
      totalPatients: result.data.res.data.length,
      patientsWithMediumOrHighRisk: n,
      MediumOrHighRiskRate: `${(n / result.data.res.data.length) * 100} %`,
      avgAge: sumAge / n,
      avgExams: sumExams / n,
      avgChildren: sumChildren / n,
    };

    res.status(200).send(processedData);
  } catch (err) {
    console.log(err);
    next(err);
  }
});

app.post(`/process/:n`, async (req, res, next) => {
  const n = parseInt(req.params.n);

  for (let i = 0; i < n; i++) {
    try {
      const id = Math.random() * (MAX - MIN) + MIN;

      await axios({
        method: "GET",
        url: `http://${process.env.HOST_NAME}:${process.env.HOST_PORT}/patients/${id}`,
      });
    } catch (err) {
      console.log(err);
    }
  }

  res.status(200).end();
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
