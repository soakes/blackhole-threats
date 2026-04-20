import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const websiteRoot = path.resolve(__dirname, "..");

const requiredFiles = [
  "index.html",
  "main.js",
  "style.css",
  "vite.config.js",
];

const requiredDomIds = [
  "site-url-link",
  "nav-apt-link",
  "ui-version",
  "ui-commit",
  "ui-date",
  "ui-fingerprint",
  "apt-code-container",
  "footer-version",
  "footer-commit",
];

const requiredEnvBindings = [
  "process.env.PUBLIC_SITE_URL",
  "process.env.PUBLIC_RELEASE_VERSION",
  "process.env.PUBLIC_COMMIT",
  "process.env.PUBLIC_BUILD_DATE",
  "process.env.PUBLIC_APT_FINGERPRINT",
];

const errors = [];

function assert(condition, message) {
  if (!condition) {
    errors.push(message);
  }
}

function load(relativePath) {
  const fullPath = path.join(websiteRoot, relativePath);
  assert(existsSync(fullPath), `Missing required file: ${relativePath}`);
  return existsSync(fullPath) ? readFileSync(fullPath, "utf8") : "";
}

function checkNodeSyntax(relativePath) {
  const result = spawnSync(process.execPath, ["--check", relativePath], {
    cwd: websiteRoot,
    encoding: "utf8",
  });

  if (result.status !== 0) {
    const output = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
    errors.push(`Syntax check failed for ${relativePath}${output ? `\n${output}` : ""}`);
  }
}

function findRootRelativeLocalAssets(html) {
  const assetRefs = [...html.matchAll(/\b(?:src|href)=["']([^"']+)["']/g)].map((match) => match[1]);
  return assetRefs.filter((ref) => ref.startsWith("/") && !ref.startsWith("//"));
}

const indexHtml = load("index.html");
const mainJs = load("main.js");
const viteConfig = load("vite.config.js");

for (const relativePath of requiredFiles) {
  assert(existsSync(path.join(websiteRoot, relativePath)), `Missing required file: ${relativePath}`);
}

checkNodeSyntax("main.js");
checkNodeSyntax("vite.config.js");

assert(
  indexHtml.includes('<script type="module" src="./main.js"></script>'),
  "index.html must load the Vite entrypoint via ./main.js",
);

assert(
  mainJs.includes("import './style.css';") || mainJs.includes('import "./style.css";'),
  "main.js must import ./style.css so Vite owns stylesheet output",
);

assert(
  viteConfig.includes("base: './'") || viteConfig.includes('base: "./"'),
  "vite.config.js must keep base set to ./ for GitHub Pages subpath compatibility",
);

for (const envBinding of requiredEnvBindings) {
  assert(mainJs.includes(envBinding), `main.js must reference ${envBinding}`);
}

for (const domId of requiredDomIds) {
  assert(indexHtml.includes(`id="${domId}"`), `index.html is missing required id="${domId}"`);
}

const rootRelativeAssets = findRootRelativeLocalAssets(indexHtml);
assert(
  rootRelativeAssets.length === 0,
  `index.html contains root-relative local asset paths that break GitHub Pages subpath deploys: ${rootRelativeAssets.join(", ")}`,
);

assert(indexHtml.includes('href="#install"'), 'index.html must retain the in-page install anchor link');
assert(indexHtml.includes('id="install"'), 'index.html must retain the install section target');

if (errors.length > 0) {
  console.error("Website validation failed:\n");
  for (const error of errors) {
    console.error(`- ${error}`);
  }
  process.exit(1);
}

console.log("Website validation passed.");
