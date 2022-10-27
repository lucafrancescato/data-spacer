import express from "express";

import patientController from "../controllers/patientController.js";

const router = express.Router();

router.route("/").get(patientController.getPatients);
router.route("/:id").get(patientController.getPatient);

export default router;
