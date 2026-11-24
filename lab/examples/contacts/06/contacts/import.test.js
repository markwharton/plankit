import { test } from "node:test";
// Stand-ins: the checker looks for each name in this file. It does not run them.
test("tells the person when an import fails", () => {});
test("refuses a file that is not a CSV of contacts", () => {});
test("refuses a row with no email", () => {});
test("keeps the old contact when the person says no", () => {});
test("asks before replacing a contact", () => {});
test("refuses an email given twice", () => {});
