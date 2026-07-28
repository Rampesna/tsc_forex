import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const directory = path.dirname(fileURLToPath(import.meta.url));
const sourcePath = path.resolve(directory, "../domain/instrument.go");
const outputPath = path.join(directory, "currency_names.json");
const source = fs.readFileSync(sourcePath, "utf8");

const start = source.indexOf("var Currencies = []CurrencyDef{");
const end = source.indexOf("// goldProducts", start);
if (start < 0 || end < 0) {
  throw new Error("currency catalog block was not found");
}

const catalog = source.slice(start, end);
const entries = [...catalog.matchAll(/\{"([A-Z]{3})",\s*"([^"]+)"\}/g)];
if (entries.length === 0) {
  throw new Error("currency catalog is empty");
}

const locales = ["tr", "en", "ar", "de", "it", "es"];
const displayNames = Object.fromEntries(
  locales
    .filter((locale) => locale !== "tr")
    .map((locale) => [
      locale,
      new Intl.DisplayNames([locale], { type: "currency", fallback: "none" }),
    ]),
);

const output = {};
for (const [, iso, turkishName] of entries) {
  const names = { tr: turkishName };
  for (const locale of locales.slice(1)) {
    const localized = displayNames[locale].of(iso);
    if (!localized) {
      throw new Error(`CLDR has no ${locale} display name for ${iso}`);
    }
    names[locale] = localized;
  }
  output[`${iso}TRY`] = names;
}

fs.writeFileSync(outputPath, `${JSON.stringify(output, null, 2)}\n`, "utf8");
console.log(`generated ${entries.length} currencies × ${locales.length} locales`);
