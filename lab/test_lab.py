"""Tests for the lab's server, run against the real checker.

    python3 -m unittest -v test_lab     (from this directory)

Each test names what must hold and includes the case that must fail.
"""
import json
import os
import shutil
import tempfile
import threading
import time
import unittest
import urllib.error
import urllib.request

import lab

HERE = os.path.dirname(os.path.abspath(__file__))
CONTROL_SRC = os.path.normpath(os.path.join(HERE, "..", "control"))


class LabServer(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.work = tempfile.mkdtemp(prefix="lab-test-")
        cls.designs = os.path.join(cls.work, "designs")
        shutil.copytree(os.path.join(HERE, "designs"), cls.designs)
        cls.server, cls.library = lab.serve([os.path.join(HERE, "examples", "lab.json"), cls.designs], control_src=CONTROL_SRC)
        cls.base = "http://127.0.0.1:%d" % cls.server.server_port
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        shutil.rmtree(cls.work, ignore_errors=True)

    def call(self, path, body=None, headers=None):
        """Returns (status, the JSON answer)."""
        data = json.dumps(body).encode() if body is not None else None
        h = {"Content-Type": "application/json", lab.HEADER: "1"} if body is not None and headers is None else (headers or {})
        req = urllib.request.Request(self.base + path, data=data, headers=h)
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                return r.status, json.loads(r.read())
        except urllib.error.HTTPError as e:
            with e:
                return e.code, json.loads(e.read())

    def scenario(self, label):
        _, got = self.call("/api/scenarios")
        found = [s for s in got["scenarios"] if s["label"] == label]
        self.assertEqual(len(found), 1, "one scenario labelled %r among %s" % (label, [s["label"] for s in got["scenarios"]]))
        return found[0]

    def verdicts(self, res):
        return [(rv["run"], f["place"], f["verdict"]) for rv in res["model"]["runs"] for f in rv["faults"]]

    def test_the_model_is_the_checkers(self):
        """The lab adds no verdict: stage 4 reads as the checker gives it."""
        sc = self.scenario("contacts 4: split the step into two parts")
        status, res = self.call("/api/model?id=" + sc["id"])
        self.assertEqual(status, 200)
        self.assertTrue(res["ok"])
        self.assertEqual(self.verdicts(res), [("import", "csv", "lost"), ("import", "rows", "lost"), ("import", "book", "lost"), ("import", "message", "output")])
        self.assertEqual(res["check"]["exit"], 2)
        self.assertIn("rejects-declared  import rejects: parse passes its refusal to import, which has no rejects; store passes its refusal to import, which has no rejects", res["check"]["lines"])

    def test_an_edit_is_checked_and_nothing_is_written(self):
        """Giving import a rejects turns lost into refused, and the file on disk is as it was."""
        sc = self.scenario("contacts 4: split the step into two parts")
        _, src = self.call("/api/source?id=" + sc["id"])
        design = json.loads(src["text"])
        next(e for e in design["elements"] if e["name"] == "import")["rejects"] = {"reason": "the import failed", "to": "message"}
        status, res = self.call("/api/check", {"id": sc["id"], "text": json.dumps(design)})
        self.assertEqual(status, 200)
        self.assertEqual([v for _, _, v in self.verdicts(res)], ["refused", "refused", "refused", "output"])
        self.assertEqual(res["model"]["definition"], "contacts.json")
        _, again = self.call("/api/source?id=" + sc["id"])
        self.assertEqual(again["text"], src["text"])
        _, saved = self.call("/api/model?id=" + sc["id"])
        self.assertEqual(self.verdicts(saved)[0], ("import", "csv", "lost"))

    def test_text_the_checker_refuses_is_reported_with_its_reason(self):
        sc = self.scenario("contacts 4: split the step into two parts")
        _, src = self.call("/api/source?id=" + sc["id"])
        design = json.loads(src["text"])
        design["colour"] = "blue"
        status, res = self.call("/api/check", {"id": sc["id"], "text": json.dumps(design)})
        self.assertEqual(status, 200)
        self.assertFalse(res["ok"])
        self.assertTrue(any("colour" in r for r in res["refused"]), res["refused"])
        self.assertTrue(all("control-lab-" not in r for r in res["refused"]), "a reason names the lab's scratch file: %s" % res["refused"])
        _, res = self.call("/api/check", {"id": sc["id"], "text": "{ not json"})
        self.assertFalse(res["ok"])

    def test_a_change_without_the_labs_header_is_refused(self):
        sc = self.scenario("contacts/contacts.json")
        status, res = self.call("/api/check", {"id": sc["id"], "text": "{}"}, headers={"Content-Type": "application/json"})
        self.assertEqual(status, 403)
        status, res = self.call("/api/save", {"id": sc["id"], "text": "{}"}, headers={"Content-Type": "application/json", lab.HEADER: "1", "Origin": "http://example.com"})
        self.assertEqual(status, 403)
        status, _ = self.call("/api/scenarios", headers={"Host": "example.com"})
        self.assertEqual(status, 403)

    def test_save_writes_only_where_it_may(self):
        readonly = self.scenario("pk itself")
        status, res = self.call("/api/save", {"id": readonly["id"], "text": "{}"})
        self.assertEqual(status, 403, res)
        sc = self.scenario("contacts/contacts.json")
        _, src = self.call("/api/source?id=" + sc["id"])
        for bad in ["../escape.json", "sub/x.json", "x.txt", ".hidden.json", "x.accepted.json", ""]:
            status, res = self.call("/api/save", {"id": sc["id"], "text": src["text"], "name": bad})
            self.assertEqual(status, 400, "%r: %s" % (bad, res))
        self.assertFalse(os.path.exists(os.path.join(self.work, "escape.json")))
        status, res = self.call("/api/save", {"id": sc["id"], "text": src["text"], "name": "variant.json"})
        self.assertEqual(status, 200, res)
        self.assertTrue(os.path.isfile(os.path.join(self.designs, "contacts", "variant.json")))
        self.assertEqual(self.scenario("contacts/variant.json")["id"], res["id"])
        status, res = self.call("/api/save", {"id": sc["id"], "text": src["text"], "name": "variant.json"})
        self.assertEqual(status, 409, res)

    def test_save_then_the_stamp_and_the_model_change(self):
        sc = self.scenario("contacts/contacts.json")
        _, src = self.call("/api/source?id=" + sc["id"])
        _, before = self.call("/api/model?id=" + sc["id"])
        design = json.loads(src["text"])
        del next(e for e in design["elements"] if e["name"] == "import")["rejects"]
        time.sleep(0.01)
        status, _ = self.call("/api/save", {"id": sc["id"], "text": json.dumps(design, indent=2)})
        self.assertEqual(status, 200)
        self.assertNotEqual(self.scenario("contacts/contacts.json")["stamp"], sc["stamp"])
        _, after = self.call("/api/model?id=" + sc["id"])
        self.assertEqual(self.verdicts(before)[0], ("import", "csv", "refused"))
        self.assertEqual(self.verdicts(after)[0], ("import", "csv", "lost"))
        self.call("/api/save", {"id": sc["id"], "text": src["text"]})

    def test_a_hold_file_is_watched(self):
        """Writing the test a hold names changes the scenario's stamp, so the page checks again."""
        sc = self.scenario("contacts/contacts.json")
        path = os.path.join(self.designs, "contacts", "contacts", "import.test.js")
        self.assertTrue(os.path.isfile(path))
        with open(path, "a") as f:
            f.write("// touched\n")
        self.assertNotEqual(self.scenario("contacts/contacts.json")["stamp"], sc["stamp"])

    def test_every_bundled_design_is_read(self):
        """Each example gives a model, and the only refusals are none."""
        _, got = self.call("/api/scenarios")
        self.assertGreaterEqual(len(got["scenarios"]), 16)
        for sc in got["scenarios"]:
            status, res = self.call("/api/model?id=" + sc["id"])
            self.assertEqual(status, 200, sc["label"])
            self.assertTrue(res["ok"], "%s: %s" % (sc["label"], res.get("refused")))
            self.assertGreaterEqual(len(res["model"]["runs"]), 1)

    def test_a_design_you_can_run_is_played_by_the_new_engine(self):
        sc = self.scenario("notes/notes.design")
        _, res = self.call("/api/model?id=" + sc["id"])
        self.assertTrue(res["ok"])
        self.assertEqual(res["model"]["labels"]["break"], "Wrong")
        self.assertEqual(sorted((f["place"], f["verdict"]) for f in res["model"]["runs"][0]["faults"]), [("the note is empty", "refused"), ("the site is down", "refused")])
        _, src = self.call("/api/source?id=" + sc["id"])
        text = src["text"].replace("        REFUSE WHEN the note is empty\n", "")
        self.assertNotEqual(text, src["text"])
        _, res = self.call("/api/check", {"id": sc["id"], "text": text})
        self.assertIn(("the note is empty", "passed"), [(f["place"], f["verdict"]) for f in res["model"]["runs"][0]["faults"]])
        self.assertTrue(any(l.startswith("unanswered  the note is empty") for l in res["check"]["lines"]))
        _, res = self.call("/api/check", {"id": sc["id"], "text": text + "\nCOLOUR blue\n"})
        self.assertFalse(res["ok"])
        self.assertIn("COLOUR", res["refused"][0])

    def test_a_design_written_as_text_is_listed_played_refused_formatted_and_saved(self):
        with open(os.path.join(self.designs, "notes", "notes.design"), encoding="utf-8") as f:
            text = f.read()
        path = os.path.join(self.designs, "notes", "written.design")
        with open(path, "w", encoding="utf-8") as f:
            f.write(text)
        with open(os.path.join(self.designs, "notes", "written.json"), "w", encoding="utf-8") as f:
            json.dump(lab.engine.read_text(text, "notes.design"), f)              # the same design, in JSON
        try:
            sc = self.scenario("notes/written.design")
            _, res = self.call("/api/model?id=" + sc["id"])
            self.assertTrue(res["ok"], res.get("refused"))
            _, json_res = self.call("/api/model?id=" + self.scenario("notes/written.json")["id"])
            self.assertEqual(self.verdicts(res), self.verdicts(json_res))           # written as text, it plays as its JSON
            broken = text + "\nFACT WHEN it rains\n"
            _, res = self.call("/api/check", {"id": sc["id"], "text": broken})
            self.assertEqual(res["refused"], ["written.design:%d: a name was expected after FACT, and WHEN is a keyword" % broken.count("\n")])      # named as the lab shows it
            _, res = self.call("/api/format", {"id": sc["id"], "text": text.replace("\n\n", "\n\n\n")})
            self.assertEqual(res["text"], text)
            _, res = self.call("/api/save", {"id": sc["id"], "text": text, "name": "copy.design"})
            self.assertEqual(res["saved"], "copy.design")
            self.scenario("notes/copy.design")
        finally:
            for name in ("written.design", "written.json", "copy.design"):
                if os.path.exists(os.path.join(self.designs, "notes", name)):
                    os.remove(os.path.join(self.designs, "notes", name))

    def test_a_runnable_design_sends_summaries_and_a_whole_trace_on_request(self):
        sc = self.scenario("notes/notes.design")
        _, res = self.call("/api/model?id=" + sc["id"])
        rv = res["model"]["runs"][0]
        for t in rv["traces"]:
            self.assertFalse({"start", "end"} & set(t), t)                     # a summary: the page asks for the rest
            self.assertTrue(all(set(s) == {"element", "outcome"} for s in t["steps"]), t["steps"])
        status, whole = self.call("/api/trace", {"id": sc["id"], "text": None, "run": rv["run"], "n": 1})
        self.assertEqual(status, 200, whole)
        self.assertIn("start", whole["trace"])
        self.assertIn("end", whole["trace"])
        self.assertEqual([(s["element"], s["outcome"]) for s in whole["trace"]["steps"]], [(s["element"], s["outcome"]) for s in rv["traces"][1]["steps"]])
        self.assertEqual(whole["trace"]["refusals"], rv["traces"][1]["refusals"])
        status, got = self.call("/api/trace", {"id": sc["id"], "text": None, "run": rv["run"], "n": len(rv["traces"])})
        self.assertEqual(status, 400)
        self.assertIn("no trace %d" % len(rv["traces"]), json.dumps(got))

    def test_a_named_fact_is_listed_with_its_meaning(self):
        sc = self.scenario("notes/notes.design")
        _, src = self.call("/api/source?id=" + sc["id"])
        text = src["text"].replace("FAULT the site is down\n", "FACT publishing MEANS NOT preview\n    SAYS the note is going to the site\nFAULT the site is down\n")
        text = text.replace("    ACTION put on site\n", "    ACTION put on site WHEN publishing\n")
        _, res = self.call("/api/check", {"id": sc["id"], "text": text})
        self.assertTrue(res["ok"], res.get("refused"))
        got = [f for f in res["model"]["facts"] if f["name"] == "publishing"]
        self.assertEqual(got, [{"name": "publishing", "fault": False, "open": False, "about": "", "says": "the note is going to the site", "named": True, "means": "not preview"}])
        self.assertTrue(all(not f.get("named") for f in res["model"]["facts"] if f["name"] != "publishing"))

    def test_the_description_prompt_is_read_from_the_readme(self):
        _, got = self.call("/api/prompts")
        text = got["description"]["text"]
        self.assertIn("<the step>", text)
        self.assertIn("<path>", text)
        self.assertEqual([p["title"] for p in got["runnable"]], ["Write a design from what a system should do", "Describe what the code does"])
        self.assertIn("--against", got["runnable"][1]["text"])
        with open(os.path.join(CONTROL_SRC, "README.md"), encoding="utf-8") as f:
            self.assertIn(text, f.read())

    def test_a_connection_the_browser_drops_prints_nothing(self):
        """A browser opens spare connections and resets them. That is not an error to report."""
        import contextlib, io, socket, struct
        err = io.StringIO()
        with contextlib.redirect_stderr(err):
            s = socket.create_connection(("127.0.0.1", self.server.server_port))
            s.setsockopt(socket.SOL_SOCKET, socket.SO_LINGER, struct.pack("ii", 1, 0))
            time.sleep(0.2)
            s.close()
            time.sleep(0.5)
            status, _ = self.call("/api/scenarios")
        self.assertEqual(status, 200)
        self.assertEqual(err.getvalue(), "")

    def test_the_page_is_served_and_loads_nothing_from_elsewhere(self):
        with urllib.request.urlopen(self.base + "/", timeout=10) as r:
            page = r.read().decode()
        self.assertIn("<title>control lab</title>", page)
        for needle in ["http://", "https://", "<script src", "<link "]:
            self.assertNotIn(needle, page.replace("http://www.w3.org", ""), needle)


class Unit(unittest.TestCase):
    def test_a_page_without_view_data_is_an_error_that_says_so(self):
        with self.assertRaises(lab.LabError) as cm:
            lab.view_model("<html><script>1</script></html>")
        self.assertIn("view-data", str(cm.exception))

    def test_files_that_moved_while_the_lab_ran_are_named_with_the_fix(self):
        with self.assertRaises(lab.LabError) as cm:
            lab.Library([os.path.join(HERE, "no-such", "lab.json")], None).scenarios()
        self.assertIn("start it again", str(cm.exception))

    def test_a_missing_checker_names_the_fix(self):
        with self.assertRaises(lab.LabError) as cm:
            lab.Checker(None, os.path.join(HERE, "examples"))
        self.assertIn("main.go", str(cm.exception))


if __name__ == "__main__":
    unittest.main()
