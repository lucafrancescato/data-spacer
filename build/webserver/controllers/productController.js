import fs from "fs";
import { dirname } from "path";
import { fileURLToPath } from "url";

const currentDir = dirname(fileURLToPath(import.meta.url));

const products = JSON.parse(fs.readFileSync(`${currentDir}/products.json`));

const getProducts = async (req, res, next) => {
  try {
    res.status(200).json({
      res: {
        length: products.length,
        data: products,
      },
    });
  } catch (err) {
    next(err);
  }
};

const getProduct = async (req, res, next) => {
  const id = req.params.id;

  const prod = products.find((el) => el.id == id);
  if (!prod)
    return next({
      message: "Il prodotto specificato non è stato trovato",
      status: 404,
    });

  try {
    res.status(200).json({
      res: {
        data: prod,
      },
    });
  } catch (err) {
    next(err);
  }
};

export default {
  getProducts,
  getProduct,
};
