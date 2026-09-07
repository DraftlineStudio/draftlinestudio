#!/usr/bin/env node
// Writes the release version into wails/wails.json (info.productVersion) so
// the Windows exe metadata and NSIS installer carry the real version instead
// of the Wails default "1.0.0". Used by .github/workflows/release.yml; the
// change is build-time only and is never committed. The version itself lives
// in wails/app.go (see docs/VERSION.md).
const fs = require("fs");
const path = require("path");

const version = process.argv[2];
if (!/^\d+\.\d+\.\d+$/.test(version || "")) {
  console.error(`usage: stamp-version.js MAJOR.MINOR.BUILD (got ${JSON.stringify(version)})`);
  process.exit(1);
}

const file = path.join(__dirname, "..", "..", "wails", "wails.json");
const project = JSON.parse(fs.readFileSync(file, "utf8"));
project.info = { ...(project.info || {}), productVersion: version };
fs.writeFileSync(file, JSON.stringify(project, null, 2) + "\n");
console.log(`stamped ${version} into ${path.relative(process.cwd(), file)}`);
