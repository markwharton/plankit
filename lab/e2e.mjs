// End-to-end test of the lab's page: a real server, the real checker and
// a headless Chrome, driven over the DevTools protocol. Needs Node 22 or
// later (for the built-in WebSocket) and Google Chrome.
//
//   node e2e.mjs            runs the checks and prints one line for each
//   node e2e.mjs shots/     also saves a screenshot of each view there
import { spawn } from "node:child_process";
import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const CHROME = process.env.CHROME || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const shots = process.argv[2];
const work = mkdtempSync(join(tmpdir(), "lab-e2e-"));
const designs = join(work, "designs");
cpSync(join(HERE, "designs"), designs, { recursive: true });
const LAB = 8791, CDP = 9391;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const procs = [];
let failed = 0;
const check = (name, ok, detail) => { if (!ok) failed++; console.log((ok ? "ok    " : "FAIL  ") + name + (ok || !detail ? "" : "\n        " + detail)); };

async function until(fn, what, ms = 8000) {
  const end = Date.now() + ms;
  for (;;) { const v = await fn(); if (v) return v; if (Date.now() > end) throw new Error("timed out waiting for " + what); await sleep(100); }
}

async function main() {
  procs.push(spawn("python3", [join(HERE, "lab.py"), "--no-open", "--port", String(LAB), join(HERE, "examples", "lab.json"), designs], { stdio: "ignore" }));
  procs.push(spawn(CHROME, ["--headless=new", "--disable-gpu", "--remote-debugging-port=" + CDP, "--user-data-dir=" + join(work, "chrome"), "--window-size=1500,1300", "about:blank"], { stdio: "ignore" }));
  await until(() => fetch("http://127.0.0.1:" + LAB + "/api/scenarios").then((r) => r.ok, () => false), "the lab's server", 60000);
  const tab = await until(() => fetch("http://127.0.0.1:" + CDP + "/json/new?http://127.0.0.1:" + LAB + "/", { method: "PUT" }).then((r) => r.json(), () => null), "Chrome");
  const ws = new WebSocket(tab.webSocketDebuggerUrl);
  await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej; });
  let seq = 0; const waiting = new Map(), errors = [];
  ws.onmessage = (m) => {
    const d = JSON.parse(m.data);
    if (d.id && waiting.has(d.id)) { waiting.get(d.id)(d); waiting.delete(d.id); }
    if (d.method === "Runtime.exceptionThrown") errors.push(d.params.exceptionDetails.exception?.description || d.params.exceptionDetails.text);
    if (d.method === "Runtime.consoleAPICalled" && d.params.type === "error") errors.push(d.params.args.map((a) => a.value || a.description).join(" "));
  };
  const send = (method, params = {}) => new Promise((res) => { const id = ++seq; waiting.set(id, res); ws.send(JSON.stringify({ id, method, params })); });
  await send("Runtime.enable"); await send("Page.enable");
  const js = async (expr) => {
    const r = await send("Runtime.evaluate", { expression: "(async () => { " + expr + " })()", awaitPromise: true, returnByValue: true });
    if (r.result.exceptionDetails) throw new Error(r.result.exceptionDetails.exception?.description || r.result.exceptionDetails.text);
    return r.result.result.value;
  };
  const text = (sel) => js("const e = document.querySelector(" + JSON.stringify(sel) + "); return e ? e.textContent : null;");
  const click = (sel, label) => js("const b = [...document.querySelectorAll(" + JSON.stringify(sel) + ")].find((x) => " + (label === undefined ? "true" : "x.textContent.trim().startsWith(" + JSON.stringify(label) + ")") + "); if (!b) throw new Error('no " + sel + " " + (label || "") + "'); b.click();");
  const open = async (label) => {
    await js("const s = document.getElementById('scn'); const o = [...s.options].find((x) => x.textContent === " + JSON.stringify(label) + "); if (!o) throw new Error('no design " + label + "'); s.value = o.value; s.dispatchEvent(new Event('change'));");
    await loaded(label);
  };
  // A design is open once the page marks it so: the select changes before the files arrive.
  const loaded = (label) => until(() => js("const s = document.getElementById('scn'); return s.selectedOptions[0] && s.selectedOptions[0].textContent === " + JSON.stringify(label) + " && document.getElementById('cols').dataset.open === s.value && document.getElementById('state').textContent === 'Live: as saved';"), "the design " + label);
  const type = (value) => js("const e = document.getElementById('ed'); e.value = " + JSON.stringify(value) + "; e.dispatchEvent(new Event('input'));");
  const shot = async (name) => {
    if (!shots) return;
    mkdirSync(shots, { recursive: true });
    const r = await send("Page.captureScreenshot", { format: "png", captureBeyondViewport: true });
    writeFileSync(join(shots, name + ".png"), Buffer.from(r.result.data, "base64"));
  };

  await until(() => js("return !!document.querySelector('#stage svg, #stage table')"), "the first drawing", 30000);
  check("the page draws a design on load", true);

  // A fault, in each of the five views.
  await open("contacts 4: split the step into two parts");
  await click("#c-brk button", "csv");
  check("breaking csv at stage 4 reads lost", /Verdict: lost/.test(await text("#readout")), await text("#readout"));
  await click("#c-view button", "Rail");
  check("Rail marks where the refusal stops", /the refusal stops here and is reported nowhere/.test(await text("#stage")));
  await shot("rail");
  await click("#c-view button", "Bus");
  check("Bus draws the playhead at the end", (await text("#controls")).includes("2 / 2"), await text("#controls"));
  await js("const r = document.getElementById('b-t'); r.value = 0; r.dispatchEvent(new Event('input'));");
  check("Bus scrubbed to the start says so", /Before the first leaf/.test(await text("#controls")));
  await click("#b-play");
  await until(async () => (await text("#controls")).includes("2 / 2"), "Play to reach the end");
  check("Bus Play runs to the end", /store does not run/.test(await text("#controls")), await text("#controls"));
  await shot("bus");
  await click("#c-view button", "Ring");
  check("Ring draws a wire for each read and write", /6 reads and writes/.test(await text("#stage")), (await text("#stage")).slice(-60));
  await js("document.querySelector('#ring-svg [data-hover=\"book\"]').dispatchEvent(new MouseEvent('click', { bubbles: true }));");
  check("Ring: clicking a place describes it", /book: lasting, owned by import/.test(await text("#controls")), await text("#controls"));
  await shot("ring");
  await click("#c-view button", "Grid");
  check("Grid shows the lost refusal on import", /refusal lost here/.test(await text("#stage")));
  await shot("grid");
  await click("#c-view button", "Matrix");
  await click("#stage button[data-place=\"rows\"]");
  check("Matrix: clicking a row selects that fault", /^rows holds a row with no email/.test(await text("#readout")), await text("#readout"));
  await shot("matrix");

  // An or with three paths, and a larger design.
  await open("save trial");
  await click("#c-view button", "Rail");
  check("the save trial offers its three paths", (await js("return document.querySelectorAll('#c-path button').length")) === 3);
  await click("#c-path button", "save → saving, store → save as");
  await click("#c-brk button", "typed name");
  check("save as refuses an empty name, up to message", /save as refuses it, through store, saving, save, and the refusal reaches message/.test(await text("#readout")), await text("#readout"));
  await open("pk itself");
  for (const v of ["Rail", "Bus", "Ring", "Grid", "Matrix"]) { await click("#c-view button", v); await sleep(60); }
  check("every view draws pk", (await js("return document.querySelectorAll('#stage table.m tr').length")) > 10);
  check("a read-only design cannot be saved", (await js("const b = document.getElementById('b-save'); return b.disabled && b.textContent === 'Read-only'")));

  // Editing: a draft is checked, compared, refused, reverted and saved.
  await open("contacts/contacts.json");
  await click("#c-view button", "Rail");
  const saved = readFileSync(join(designs, "contacts", "contacts.json"), "utf8");
  const edited = JSON.parse(saved);
  delete edited.elements.find((e) => e.name === "import").rejects;
  await type(JSON.stringify(edited, null, 2));
  await until(async () => (await text("#state")) === "Draft: not saved" && /lost/.test(await text("#c-brk")), "the draft to be checked");
  check("an edit is checked without being saved", readFileSync(join(designs, "contacts", "contacts.json"), "utf8") === saved);
  await js("const s = document.getElementById('cmp'); s.value = 'disk'; s.dispatchEvent(new Event('change'));");
  await until(async () => /sweep verdicts changed/.test(await text("#panel") || ""), "the compare panel");
  const cmp = await text("#panel");
  check("Compare lists the new finding and the changed verdicts", /new.*rejects-declared/.test(cmp) && /3 sweep verdicts changed, 4 the same/.test(cmp), cmp.slice(0, 400));
  check("the top bar shows the change in findings", /▲/.test(await text("#stats")), await text("#stats"));
  await shot("compare");
  await type(JSON.stringify(Object.assign({ colour: "blue" }, edited)));
  await until(async () => (await text("#state")) === "The checker refuses this text", "the refusal");
  check("text the checker refuses shows its reason", /colour/.test(await text("#edstatus")), await text("#edstatus"));
  check("the views keep the last accepted version", /lost/.test(await text("#c-brk")));
  await click("#b-rev");
  await until(async () => (await text("#state")) === "Live: as saved", "the revert");
  check("Revert returns to the saved file", !/lost/.test(await text("#c-brk")));

  await type(JSON.stringify(edited, null, 2));
  await until(async () => (await text("#state")) === "Draft: not saved", "the draft");
  await js("document.getElementById('saveas').value = '../outside.json';");
  await click("#b-saveas");
  await until(async () => /name is letters/.test(await text("#toast") || ""), "the refusal of a bad name");
  check("a name with a directory is refused and nothing is written", !existsSync(join(designs, "outside.json")) && !existsSync(join(work, "outside.json")));
  await js("document.getElementById('saveas').value = 'no-rejects.json';");
  await click("#b-saveas");
  await loaded("contacts/no-rejects.json");
  check("Save as new writes the file and opens it", existsSync(join(designs, "contacts", "no-rejects.json")) && /lost/.test(await text("#c-brk")));

  // A change made outside the page, as a session or an editor would make it.
  writeFileSync(join(designs, "contacts", "no-rejects.json"), saved);
  await until(async () => !/lost/.test(await text("#c-brk")), "the page to follow the file on disk", 10000);
  check("a file changed on disk is drawn again without a click", (await text("#state")) === "Live: as saved");

  await click("#c-panel button", "Prompts");
  await until(async () => /Write a description file/.test(await text("#p-desc-text") || ""), "the description prompt");
  const prompt = await text("#p-desc-text");
  check("the description prompt names the step and the file", /one\s+element of this repository: import\./.test(prompt) && /definition file\s+no-rejects\.json/.test(prompt.replace(/\n/g, " ")), prompt);
  await open("contacts 5: say where the refusal lands");
  await click("#c-panel button", "Tests to write");
  check("Tests to write lists the six rules with no hold", (await js("return document.querySelectorAll('#panel .entry').length")) === 6);
  await shot("tests");

  // A design you can run: the same page and views, fed by the new engine.
  await open("pk, as a design you can run");
  check("a runnable design labels its controls Given and Wrong", (await text("#l-path")) === "Given" && (await text("#l-brk")) === "Wrong");
  for (const v of ["Rail", "Bus", "Ring", "Grid", "Matrix"]) { await click("#c-view button", v); await sleep(80); }
  check("every view draws the runnable pk", (await js("return document.querySelectorAll('#stage table.m tr').length")) > 20);
  await shot("runnable-matrix");
  await click("#c-run button", "release");
  await js("const s = document.querySelector('#s-path'); s.value = [...s.options].find((o) => o.text === 'given → the last commit is a release commit').value; s.dispatchEvent(new Event('change'));");   // release has more than eight choices, so they are a list
  await click("#c-brk button", "origin rejects the push");
  check("a fault of a runnable design reads as the engine played it", /Verdict: refused/.test(await text("#readout")) && /push in merge flow stops the run/.test(await text("#readout")), await text("#readout"));
  await click("#c-view button", "Bus");
  await shot("runnable-bus");
  await click("#c-view button", "Rail");
  await shot("runnable-rail");
  await click("#c-view button", "Grid");
  await click("#g-a");
  const answers = await text("#stage");
  check("Grid shows who answers what", /working tree holds what is kept nowhere/.test(answers) && /refuse/.test(answers) && /warn/.test(answers) && /accept/.test(answers), answers.slice(0, 200));
  await shot("runnable-grid-answers");
  await click("#c-view button", "Ring");
  check("Ring draws facts on a third arc", /facts: a red ring is a fault/.test(await text("#stage")) && (await js("return document.querySelectorAll('#ring-svg [data-hover=\"origin rejects the push\"]').length")) === 1);
  await js("document.querySelector('#ring-svg [data-hover=\"origin rejects the push\"]').dispatchEvent(new MouseEvent('click', { bubbles: true }));");
  check("Ring: clicking a fact says who answers it", /Answered by push in trunk flow \(refuse\)/.test(await text("#controls")), await text("#controls"));
  await js("document.querySelector('#ring-svg [data-hover=\"a release is pending\"]').dispatchEvent(new MouseEvent('click', { bubbles: true }));");
  check("Ring: clicking a named fact says what it means", /Named for: the last commit is a release commit and not the release is tagged and not the release tag is on another commit/.test(await text("#controls")), await text("#controls"));
  await shot("runnable-ring");
  await click("#c-view button", "Rail");
  await click("#c-run button", "ship");
  check("many choices of what is given become a list", (await js("return document.querySelectorAll('#s-path option').length")) > 8);
  await click("#c-run button", "release");
  await click("#c-panel button", "Prompts");
  await until(async () => /Describe what the code does/.test(await text("#panel") || ""), "the runnable prompts");
  check("a runnable design offers its own prompts", /Write a design from what a system should do/.test(await text("#panel")));
  await click("#c-panel button", "Findings");
  check("a design with no questions says so", /Questions for the owner/.test(await text("#panel")) && /No findings\./.test(await text("#panel")), await text("#panel"));
  await open("pk on git, written blind from its pages");
  await click("#c-panel button", "Findings");
  check("its questions are listed", /unanswered/.test(await text("#panel")) && /Questions for the owner/.test(await text("#panel")));
  await open("notes/notes.design");
  const note = readFileSync(join(designs, "notes", "notes.design"), "utf8");
  await type(note.replace("        REFUSE WHEN the note is empty\n", ""));
  await until(async () => (await text("#state")) === "Draft: not saved" && /passed/.test(await text("#c-brk")), "the runnable draft to be played");
  await click("#c-brk button", "the note is empty");
  check("an edit to a runnable design is played at once", /nothing answers it/.test(await text("#readout")) && /draft, site/.test(await text("#readout")), await text("#readout"));
  await type(note + "\nCOLOUR blue\n");
  await until(async () => (await text("#state")) === "The checker refuses this text", "the engine's refusal");
  check("the engine's reason for refusing is shown, with its line", /notes\.design:\d+: .*COLOUR/.test(await text("#edstatus")), await text("#edstatus"));
  await type(note.replace("\n\n", "\n\n\n\n"));
  await click("#b-fmt");
  await until(async () => (await js("return document.getElementById('ed').value")) === note, "the design to be formatted");
  check("Format writes a design as the printer does", true);
  await click("#b-rev");

  check("the page threw no error", errors.length === 0, errors.join("\n        "));
  ws.close();
}

main().catch((e) => { failed++; console.log("FAIL  " + e.message); }).finally(async () => {
  await Promise.all(procs.map((p) => new Promise((done) => { if (p.exitCode !== null) return done(); p.once("exit", done); p.kill(); })));
  try { rmSync(work, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 }); }
  catch (e) { console.log("note  the scratch directory " + work + " was left behind: " + e.code); }
  console.log(failed ? failed + " failed" : "all passed");
  process.exit(failed ? 1 : 0);
});
