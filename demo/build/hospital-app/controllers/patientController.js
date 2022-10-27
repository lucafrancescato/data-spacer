import fs from "fs";
import { dirname } from "path";
import { fileURLToPath } from "url";

const currentDir = dirname(fileURLToPath(import.meta.url));

const patients = JSON.parse(fs.readFileSync(`${currentDir}/patients.json`));

const getPatients = async (req, res, next) => {
  try {
    res.status(200).json({
      res: {
        length: patients.length,
        data: patients,
      },
    });
  } catch (err) {
    next(err);
  }
};

const getPatient = async (req, res, next) => {
  const id = req.params.id;

  const patient = patients.find((el) => el.id == id);
  if (!patient)
    return next({
      message: "Il paziente specificato non è stato trovato",
      status: 404,
    });

  try {
    res.status(200).json({
      res: {
        data: patient,
      },
    });
  } catch (err) {
    next(err);
  }
};

export default {
  getPatients,
  getPatient,
};
