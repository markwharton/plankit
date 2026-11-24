"""Tests of the engine on small designs.

    python3 -m unittest test_engine     (from this directory)

Each test has the design that must be reported and the one that must not.
"""
import copy
import unittest

import engine


def notes():
    """Publishing a note: save a draft, then put it on the site."""
    return {
        "system": "notes", "reports": "message",
        "places": [
            {"name": "text", "kind": "outside"},
            {"name": "draft", "kind": "lasting"},
            {"name": "site", "kind": "lasting"},
            {"name": "message", "kind": "report"},
        ],
        "facts": [
            {"name": "preview", "says": "--preview is given"},
            {"name": "the note is empty", "fault": True, "about": "text"},
            {"name": "the site is down", "fault": True, "says": "the site does not answer"},
        ],
        "parts": {"publish": {"seq": [
            {"do": "save draft", "reads": ["text"], "changes": [{"place": "draft", "later": "undone"}],
             "handles": [{"when": "the note is empty", "then": "refuse"}]},
            {"do": "put on site", "reads": ["draft"], "changes": ["site"],
             "handles": [{"when": "the site is down", "then": "refuse"}]},
        ]}},
        "runs": ["publish"],
        "rules": [{"rule": "a preview shows the note as it will appear", "about": "put on site", "when": "preview"}],
    }


def book():
    """An address book: import contacts from a file, each under its email."""
    return {
        "system": "contacts", "reports": "message",
        "places": [{"name": "file", "kind": "outside"}, {"name": "book", "kind": "lasting", "keyed": True}, {"name": "message", "kind": "report"}],
        "facts": [{"name": "the file is not a list of contacts", "fault": True, "about": "file"}],
        "parts": {"import": {"do": "import", "reads": ["file"], "changes": [{"place": "book", "how": "create", "many": True, "later": "stands", "because": "an import is not undone"}],
                             "handles": [{"when": "the file is not a list of contacts", "then": "refuse"}]}},
        "runs": ["import"],
    }


def step(doc, name):
    return engine.Design(doc).actions[name]


def kinds(doc, root="."):
    d = engine.Design(doc, root)
    return [(f["kind"], f["subject"]) for f in engine.findings(d, engine.explore_all(d))]


def outcomes(doc):
    d = engine.Design(doc)
    return sorted((tuple(p.given()), p.outcome()) for p in engine.explore_all(d))


class Engine(unittest.TestCase):
    def test_the_design_as_it_is_leaves_no_question(self):
        self.assertEqual(kinds(notes()), [])
        self.assertEqual(outcomes(notes()), [((), "completed"), (("the note is empty",), "refused"), (("the site is down",), "refused")])

    def test_one_thing_goes_wrong_in_a_play_unless_the_design_allows_more(self):
        self.assertTrue(all(len(g) <= 1 for g, _ in outcomes(notes())))
        doc = notes()
        doc["faults"] = 2
        doc["parts"]["publish"]["seq"][0]["handles"][0]["then"] = "warn"      # an empty note is skipped, and the run goes on
        doc["places"][1]["kind"] = "run"
        self.assertIn((("the note is empty", "the site is down"), "refused"), outcomes(doc))

    def test_a_fact_that_makes_no_difference_makes_no_extra_play(self):
        doc = notes()
        self.assertEqual(len(outcomes(doc)), 3)                    # nothing asks for preview
        doc["parts"]["publish"]["seq"][1]["when"] = {"not": "preview"}
        self.assertEqual(len(outcomes(doc)), 4)                    # put on site asks for it, and only when it is reached

    def test_a_fact_can_be_possible_only_when_another_holds(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["when"] = {"not": "preview"}
        doc["facts"][1]["only"] = {"not": "preview"}               # an empty note is not something a preview meets
        self.assertNotIn((("preview", "the note is empty"), "refused"), outcomes(doc))
        self.assertEqual(len(outcomes(doc)), 4)

    def test_something_wrong_that_nothing_answers_is_one_question_and_what_rests_on_it_is_named(self):
        doc = notes()
        del doc["parts"]["publish"]["seq"][0]["handles"]
        d = engine.Design(doc)
        fs = engine.findings(d, engine.explore_all(d))
        self.assertEqual([(f["kind"], f["subject"]) for f in fs], [("unanswered", "the note is empty")])
        self.assertEqual(fs[0]["actions"], ["save draft"])
        self.assertEqual(sorted(fs[0]["rest"]), ["draft", "site"])   # the bad note reaches the site through the draft

    def test_each_way_of_answering_does_what_it_says(self):
        for then, outcome, draft in [("refuse", "refused", "untouched"), ("warn", "completed with a warning", "untouched"),
                                     ("tell", "completed", "set"), ("accept", "completed", "set")]:
            with self.subTest(then=then):
                doc = notes()
                doc["places"][1]["kind"] = "lasting"
                doc["parts"]["publish"]["seq"][0]["handles"][0].update(then=then, because="an empty note is allowed")
                doc["parts"]["publish"]["seq"][1]["reads"] = ["text"]         # so that a skipped draft is not read
                d = engine.Design(doc)
                p = [p for p in engine.explore_all(d) if p.given() == ["the note is empty"]][0]
                self.assertEqual((p.outcome(), p.state["draft"]), (outcome, draft))
                self.assertEqual(p.state["message"], "untouched" if then == "accept" else "set")
                self.assertEqual(kinds(doc), [])                               # each is an answer

    def test_an_action_can_do_its_work_and_then_fail(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["handles"][0]["then"] = "fail"       # the site took the note and then broke
        doc["parts"]["publish"]["seq"][1]["changes"] = [{"place": "site"}]
        self.assertEqual(kinds(doc), [("left-behind", "site")])                 # its own change needs an answer too
        doc["parts"]["publish"]["seq"][1]["changes"][0]["later"] = "undone"
        self.assertEqual(kinds(doc), [])
        d = engine.Design(doc)
        p = [p for p in engine.explore_all(d) if p.given() == ["the site is down"]][0]
        self.assertEqual((p.outcome(), p.state["site"], p.state["draft"], p.steps[-1]), ("refused", "undone", "undone", ("put on site", "failed")))

    def test_what_a_design_waves_through_says_why(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["handles"][0]["then"] = "accept"
        doc["parts"]["publish"]["seq"][0]["changes"][0]["later"] = "stands"
        self.assertEqual(kinds(doc), [("unexplained", "save draft: accept, when the note is empty"), ("unexplained", "save draft: its change to draft stands")])
        doc["parts"]["publish"]["seq"][0]["handles"][0]["because"] = "an empty note is a way to clear the page"
        doc["parts"]["publish"]["seq"][0]["changes"][0]["because"] = "a draft is kept for another try"
        self.assertEqual(kinds(doc), [])

    def test_a_difference_from_the_code_can_be_accepted_with_a_reason_and_goes_stale(self):
        design, code = notes(), notes()
        del code["parts"]["publish"]["seq"][0]["handles"]
        subject = "save draft: the design refuses, the code goes ahead"
        got = engine.compare(engine.Design(design), engine.Design(code), [{"finding": subject, "because": "the code is being replaced"}])
        self.assertEqual([(f["subject"], f.get("accepted"), f.get("because")) for f in got], [(subject, True, "the code is being replaced")])
        got = engine.compare(engine.Design(design), engine.Design(notes()), [{"finding": subject, "because": "the code is being replaced"}])
        self.assertEqual([(f["kind"], f["subject"]) for f in got], [("stale", subject)])       # the code was fixed: the acceptance is a question again
        import os, tempfile
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "code.accepted.json")
            self.assertEqual(engine.read_accepted(path), [])
            with open(path, "w") as f:
                f.write('{"accepted": [{"finding": "x"}]}')
            with self.assertRaises(engine.Refused) as cm:
                engine.read_accepted(path)
            self.assertIn("because", str(cm.exception))

    def test_answers_are_tried_in_the_order_written(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["handles"] = [{"when": {"all": ["the note is empty", "preview"]}, "then": "accept"}, {"when": "the note is empty", "then": "refuse"}]
        self.assertIn((("preview", "the note is empty"), "completed"), outcomes(doc))
        doc["parts"]["publish"]["seq"][0]["handles"].reverse()
        self.assertIn((("the note is empty",), "refused"), outcomes(doc))
        self.assertNotIn((("preview", "the note is empty"), "completed"), outcomes(doc))

    def test_a_bad_value_can_be_accepted_under_a_fact(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["handles"] = [{"when": {"all": ["the note is empty", {"not": "preview"}]}, "then": "refuse"}]
        self.assertEqual(kinds(doc), [("unanswered", "the note is empty")])    # in a preview nothing answers it
        doc["parts"]["publish"]["seq"][0]["handles"].append({"when": {"all": ["the note is empty", "preview"]}, "then": "accept", "because": "a preview shows whatever is there"})
        self.assertEqual(kinds(doc), [])

    def test_asking_a_person_plays_both_answers(self):
        doc = book()
        self.assertEqual(kinds(doc), [("unanswered", "book already has the record"), ("unanswered", "book is given a key twice")])
        doc["parts"]["import"]["handles"] += [{"when": "book already has the record", "then": "ask"}, {"when": "book is given a key twice", "then": "refuse"}]
        self.assertEqual(kinds(doc), [])
        self.assertEqual(outcomes(doc), [
            ((), "completed"), (("book already has the record",), "declined"),
            (("book already has the record", "the person says yes at import"), "completed"),
            (("book is given a key twice",), "refused"), (("the file is not a list of contacts",), "refused")])
        doc["parts"]["import"]["handles"][1]["no"] = "skip"
        self.assertIn((("book already has the record",), "completed"), outcomes(doc))

    def test_an_answer_with_no_when_always_applies(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["handles"].append({"then": "ask"})      # the person is always asked before the site is changed
        self.assertEqual(kinds(doc), [])
        got = outcomes(doc)
        self.assertIn(((), "declined"), got)
        self.assertIn((("the person says yes at put on site",), "completed"), got)
        self.assertIn("put on site: ask, always", [x["subject"] for x in engine.tests(engine.Design(doc))])

    def test_an_existing_record_can_be_normal_for_a_place(self):
        doc = book()
        doc["parts"]["import"]["handles"].append({"when": "book is given a key twice", "then": "refuse"})
        self.assertEqual(kinds(doc), [("unanswered", "book already has the record")])
        doc["facts"].append({"name": "book already has the record", "about": "book", "says": "an import is run again over the same book"})
        doc["rules"] = [{"rule": "a contact already in the book is brought up to date", "about": "import", "when": "book already has the record"}]
        self.assertEqual(kinds(doc), [])                                  # a state of the world now, and it no longer uses up the one fault
        self.assertIn((("book already has the record", "the file is not a list of contacts"), "refused"), outcomes(doc))

    def test_replacing_a_record_asks_whose_it_is(self):
        doc = book()
        doc["parts"]["import"]["changes"] = [{"place": "book", "how": "replace", "later": "stands", "because": "an import is not undone"}]
        self.assertEqual(kinds(doc), [("unanswered", "book holds another's record")])
        doc["places"][1]["keyed"] = False
        with self.assertRaises(engine.Refused) as cm:
            engine.Design(doc)
        self.assertIn("not keyed", str(cm.exception))

    def test_a_change_says_what_happens_to_it_when_the_run_stops_later(self):
        doc = notes()
        del doc["parts"]["publish"]["seq"][0]["changes"][0]["later"]
        self.assertEqual(kinds(doc), [("left-behind", "draft")])
        for later, state in [("undone", "undone"), ("stands", "set")]:
            doc["parts"]["publish"]["seq"][0]["changes"][0].update(later=later, because="a draft is kept for another try")
            self.assertEqual(kinds(doc), [])
            d = engine.Design(doc)
            p = [p for p in engine.explore_all(d) if p.given() == ["the site is down"]][0]
            self.assertEqual(p.state["draft"], state)

    def test_an_expectation_can_be_about_one_failure_a_state_or_whether_an_action_runs(self):
        doc = notes()
        doc["expects"] = [
            {"name": "a preview changes nothing", "when": "preview", "never changes": ["draft", "site"]},
            {"name": "a note that fails to publish leaves no draft", "when": "the site is down", "ends": {"draft": "unchanged"}},
            {"name": "an empty note is never put on the site", "when": "the note is empty", "never runs": ["put on site"]},
            {"name": "the site is written after the draft", "before": "put on site", "place": "draft", "is": "set"},
        ]
        self.assertEqual(kinds(doc), [("expect", "a preview changes nothing")])
        d = engine.Design(doc)
        self.assertEqual(engine.findings(d, engine.explore_all(d))[0]["example"]["given"], ["preview"])
        for a in doc["parts"]["publish"]["seq"]:
            a["when"] = {"not": "preview"}
        self.assertEqual(kinds(doc), [])
        doc["parts"]["publish"]["seq"][0]["changes"][0].update(later="stands", because="a draft is kept for another try")
        self.assertEqual(kinds(doc), [("expect", "a note that fails to publish leaves no draft")])

    def test_a_change_makes_a_fact_true_for_the_rest_of_the_run(self):
        doc = notes()
        doc["facts"].append({"name": "a draft is waiting", "about": "draft", "says": "an earlier draft was never published"})
        doc["parts"]["publish"]["seq"].insert(0, {"do": "check for a draft", "reads": ["draft"], "handles": [{"when": "a draft is waiting", "then": "refuse"}]})
        doc["parts"]["publish twice"] = {"seq": ["publish", {"do": "look again", "reads": ["draft"], "handles": [{"when": {"not": "a draft is waiting"}, "then": "refuse"}]}]}
        doc["runs"] = ["publish twice"]
        last = lambda d: sorted({p.steps[-1] for p in engine.explore_all(engine.Design(d)) if not p.given()})
        self.assertEqual(last(doc), [("look again", "refused")])     # nothing says saving a draft leaves one waiting
        doc["parts"]["publish"]["seq"][1]["changes"][0]["makes"] = {"a draft is waiting": True}
        self.assertEqual(last(doc), [("look again", "ran")])

    def test_what_was_wrong_with_a_value_is_gone_once_the_run_writes_it(self):
        doc = notes()
        doc["facts"].append({"name": "the old draft is damaged", "fault": True, "about": "draft"})
        self.assertEqual(kinds(doc), [("unused", "the old draft is damaged")])   # put on site reads the draft this run wrote, so it never matters
        doc["parts"]["publish"]["seq"][0]["when"] = {"not": "preview"}
        self.assertEqual(kinds(doc), [("unanswered", "the old draft is damaged")])   # in a preview it reads the old one

    def test_an_expectation_can_name_a_fact_whose_only_names_one_no_play_asked(self):
        doc = notes()
        doc["facts"].append({"name": "a hook is set", "says": "a hook is configured"})
        doc["facts"].append({"name": "the hook fails", "fault": True, "only": "a hook is set"})
        doc["parts"]["publish"]["seq"].append({"do": "run the hook", "when": {"all": [{"not": "preview"}, "a hook is set"]}, "reads": ["site"], "handles": [{"when": "the hook fails", "then": "refuse"}]})
        doc["expects"] = [{"name": "a failed hook leaves no draft", "when": "the hook fails", "ends": {"draft": "unchanged"}}]
        self.assertEqual(kinds(doc), [("left-behind", "site")])              # no crash in a preview, where no play asked whether a hook is set
        doc["parts"]["publish"]["seq"][0]["changes"][0].update(later="stands", because="a draft is kept for another try")
        self.assertEqual(kinds(doc), [("expect", "a failed hook leaves no draft"), ("left-behind", "site")])   # and it is checked where the hook did fail
        d = engine.Design(doc)
        self.assertEqual(engine.findings(d, engine.explore_all(d))[0]["plays"], 1)                             # there, and nowhere else

    def test_which_run_it_is_is_a_fact(self):
        doc = notes()
        doc["parts"]["publish twice"] = {"seq": ["publish", {"do": "say done", "changes": ["message"]}]}
        doc["runs"].append("publish twice")
        doc["facts"][1]["only"] = "in publish"                                # an empty note matters to publish alone
        self.assertEqual([o for o in outcomes(doc) if "the note is empty" in o[0]], [(("the note is empty",), "refused")])
        d = engine.Design(doc)
        self.assertEqual({p.run for p in engine.explore_all(d) if "the note is empty" in p.given()}, {"publish"})
        self.assertTrue(all(not g.startswith("in ") for p in engine.explore_all(d) for g in p.given()))

    def test_a_fact_nothing_asks_about_is_reported(self):
        doc = notes()
        doc["facts"].append({"name": "loud", "says": "--loud is given"})
        self.assertEqual(kinds(doc), [("unused", "loud")])
        doc["rules"].append({"rule": "with --loud the message is in capitals", "about": "save draft", "when": "loud"})
        self.assertEqual(kinds(doc), [])
        del doc["rules"]
        doc["expects"] = [{"name": "a preview leaves the text alone", "when": "preview", "ends": {"text": "untouched"}}]
        self.assertEqual(kinds(doc), [("unused", "loud")])
        doc["expects"] = [{"name": "nothing is ever reported", "never changes": ["message"]}]
        d = engine.Design(doc)
        self.assertEqual(engine.findings(d, engine.explore_all(d))[0]["shows"], ["A failure is reported to message."])

    def test_what_is_not_decided_is_played_both_ways_and_asked(self):
        doc = notes()
        doc["facts"].append({"name": "a preview saves the draft", "open": True, "says": "nobody has said whether a preview saves the draft"})
        self.assertEqual(kinds(doc), [("unused", "a preview saves the draft")])        # listed, and nothing depends on it yet
        doc["parts"]["publish"]["seq"][0]["when"] = {"any": [{"not": "preview"}, "a preview saves the draft"]}
        doc["parts"]["publish"]["seq"][1]["when"] = {"not": "preview"}
        d = engine.Design(doc)
        fs = engine.findings(d, engine.explore_all(d))
        self.assertEqual([(f["kind"], f["subject"]) for f in fs], [("open", "a preview saves the draft")])
        self.assertIn("When it holds, these run and otherwise do not: save draft.", fs[0]["shows"])
        self.assertEqual({p.env["a preview saves the draft"] for p in engine.explore_all(d) if "a preview saves the draft" in p.env}, {True, False})
        doc["parts"]["publish"]["seq"][0]["when"] = {"not": "preview"}                  # the owner decides: it does not
        del doc["facts"][-1]
        self.assertEqual(kinds(doc), [])
        doc["facts"].append({"name": "odd", "open": True, "fault": True})
        with self.assertRaises(engine.Refused) as cm:
            engine.Design(doc)
        self.assertIn("not something that goes wrong", str(cm.exception))

    def test_skipping_the_writer_of_a_value_is_reported_at_its_reader(self):
        doc = notes()
        doc["places"][1]["kind"] = "run"
        self.assertEqual(kinds(doc), [])
        doc["parts"]["publish"]["seq"][0]["when"] = {"not": "preview"}
        self.assertEqual(kinds(doc), [("reads-unwritten", "put on site reads draft")])

    def test_parts_in_no_order_must_not_touch_what_another_changes(self):
        doc = notes()
        doc["parts"]["publish"] = {"all": doc["parts"]["publish"]["seq"]}
        self.assertEqual(kinds(doc), [("order", "save draft and put on site")])
        doc["parts"]["publish"]["all"][1]["reads"] = ["text"]
        self.assertEqual([k for k in kinds(doc) if k[0] == "order"], [])

    def test_a_place_is_changed_only_by_its_owner_and_never_when_it_is_outside(self):
        doc = notes()
        doc["parts"]["drafting"] = {"seq": [doc["parts"]["publish"]["seq"][0]]}
        doc["parts"]["publish"]["seq"][0] = "drafting"
        doc["places"][1]["owner"] = "drafting"
        self.assertEqual(kinds(doc), [])
        doc["places"][2]["owner"] = "drafting"
        self.assertEqual(kinds(doc), [("owner", "put on site changes site")])
        doc["places"][2] = {"name": "site", "kind": "outside"}
        self.assertEqual(kinds(doc), [("owner", "put on site changes site")])

    def test_a_design_names_where_failures_are_reported_once(self):
        doc = notes()
        del doc["reports"]
        self.assertEqual(kinds(doc), [("silent", "the design")])
        doc["reports"] = "nowhere"
        with self.assertRaises(engine.Refused):
            engine.Design(doc)

    def test_a_part_is_used_in_two_runs_without_being_copied(self):
        doc = notes()
        doc["parts"]["publish twice"] = {"seq": ["publish", {"do": "say done", "changes": ["message"]}]}
        doc["runs"].append("publish twice")
        self.assertEqual(sorted({p.run for p in engine.explore_all(engine.Design(doc))}), ["publish", "publish twice"])

    def test_the_code_is_held_to_the_design_play_for_play(self):
        design, code = notes(), notes()
        self.assertEqual(engine.compare(engine.Design(design), engine.Design(code)), [])
        del code["parts"]["publish"]["seq"][0]["handles"]             # the code saves an empty note
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], ["save draft: the design refuses, the code goes ahead"])   # one difference, one question
        self.assertIn("Every play where this happens has: the note is empty.", got[0]["shows"])
        self.assertIn("Because of it these end differently: draft, site, message.", got[0]["shows"])
        code = notes()
        code["parts"]["publish"]["seq"][0]["changes"][0].update(later="stands", because="the code keeps it")   # the code keeps the draft when the site is down
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], ["draft ends undone in the design and set in the code"])
        self.assertIn("Every play where this happens has: the site is down.", got[0]["shows"])
        code = notes()
        code["facts"].append({"name": "the disk is full", "fault": True})   # the code refuses for a reason the design never names
        code["parts"]["publish"]["seq"][0]["handles"].append({"when": "the disk is full", "then": "refuse"})
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], ["save draft: the design goes ahead, the code refuses"])
        self.assertIn("Every play where this happens has: the disk is full.", got[0]["shows"])

    def test_a_design_the_engine_cannot_play_is_refused_with_the_reason(self):
        cases = [
            (lambda d: d["places"][0].update(colour="red"), '"colour"'),
            (lambda d: d["parts"]["publish"]["seq"][0].update(reads=["nowhere"]), '"nowhere"'),
            (lambda d: d["parts"]["publish"]["seq"][0].update(when="dry run"), '"dry run"'),
            (lambda d: d["parts"]["publish"]["seq"].append("gone"), '"gone"'),
            (lambda d: d["parts"]["publish"]["seq"].append({"do": "idle"}), "reads nothing and changes nothing"),
            (lambda d: d["parts"].update(loop={"seq": ["loop"]}), "uses it in turn"),
            (lambda d: d.update(expects=[{"name": "x"}]), "says nothing to check"),
            (lambda d: d["parts"]["publish"]["seq"][0]["handles"][0].update(then="shrug"), "refuse, fail, warn, tell, ask, accept"),
            (lambda d: d["parts"]["publish"]["seq"][0]["changes"][0].update(makes={"nothing": True}), '"nothing"'),
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=[{"place": "site", "how": "create"}]), "not keyed"),
            (lambda d: d["places"][1].update(owner="nobody"), "is not a part"),
        ]
        engine.Design(notes())
        for change, reason in cases:
            with self.subTest(reason=reason):
                doc = notes()
                change(doc)
                with self.assertRaises(engine.Refused) as cm:
                    engine.Design(doc)
                self.assertIn(reason, str(cm.exception))


if __name__ == "__main__":
    unittest.main()
