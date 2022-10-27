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
app.get(`/processed-data`, async (req, res, next) => {
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
      if (!el.registration.startsWith("2022")) continue;
      sumAge += el.age;
      sumExams += el.medicalExaminations;
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
    next(err);
  }
});

app.get(`/simulate`, async (req, res, next) => {
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
