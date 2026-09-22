import { readFile, readdir } from "node:fs/promises";
import htmlhint from "htmlhint";

const { HTMLHint } = htmlhint;

const rules = JSON.parse(await readFile(".htmlhintrc", "utf8"));
let errors = 0;

for (const directory of ["src/templates", "src/static"]) {
  for (const name of await readdir(directory)) {
    if (!name.endsWith(".html")) continue;
    const file = `${directory}/${name}`;
    let source = await readFile(file, "utf8");
    if (directory === "src/templates") {
      source = source.replace(/\{\{[\s\S]*?\}\}/g, (action) =>
        action.replace(/[^\n\r]/g, " "),
      );
    }
    for (const issue of HTMLHint.verify(source, rules)) {
      console.error(`${file}:${issue.line}:${issue.col} ${issue.message} (${issue.rule.id})`);
      errors++;
    }
  }
}

if (errors) process.exitCode = 1;
else console.log("HTML checks passed.");
