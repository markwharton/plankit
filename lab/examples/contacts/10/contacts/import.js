import { openBook } from "./book.js";

export async function importContacts(file, ui) {
  try {
    const rows = parseCsv(await readFile(file));
    await storeRows(rows, openBook(), ui);
  } catch (err) {
    showMessage(ui, "Import failed: " + err.message);
  }
}

function readFile(file) {
  return file.text();
}

function parseCsv(text) {
  const [head, ...lines] = text.trim().split("\n");
  const columns = head.split(",");
  if (!columns.includes("email")) throw new Error("the file is not a CSV of contacts");
  return lines.map((line) => {
    const cells = line.split(",");
    return Object.fromEntries(columns.map((c, i) => [c, cells[i]]));
  });
}

function refuseRepeated(rows) {
  const seen = new Set();
  for (const row of rows) {
    if (seen.has(row.email)) throw new Error("the email " + row.email + " is given twice");
    seen.add(row.email);
  }
}

async function storeRows(rows, book, ui) {
  for (const row of rows) {
    if (!row.email) throw new Error("a row has no email");
  }
  refuseRepeated(rows);
  let added = 0;
  for (const row of rows) {
    if (book.has(row.email) && !(await confirmReplace(ui, row.email))) continue;
    book.set(row.email, row);
    added++;
  }
  showMessage(ui, added + " contacts added");
}

function confirmReplace(ui, email) {
  return ui.ask("Replace the contact you have for " + email + "?");
}

function showMessage(ui, text) {
  ui.message.textContent = text;
}
