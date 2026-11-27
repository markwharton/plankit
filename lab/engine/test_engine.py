"""Tests of the engine on small designs.

    python3 -m unittest test_engine     (from this directory)

Each test has the design that must be reported and the one that must not.
"""
import copy
import json
import os
import re
import tempfile
import unittest

import engine


def notes():
    """Publishing a note: save a draft, then put it on the site."""
    return {
        "system": "notes", "reports": "message",
        "places": [
            {"name": "text", "is": "one", "kind": "outside"},
            {"name": "draft", "is": "one", "kind": "lasting"},
            {"name": "site", "is": "one", "kind": "lasting"},
            {"name": "message", "is": "one", "kind": "report"},
        ],
        "facts": [
            {"name": "preview", "says": "--preview is given"},
            {"name": "the note is empty", "fault": True, "about": "text"},
            {"name": "the site is down", "fault": True, "says": "the site does not answer"},
        ],
        "parts": {"publish": {"seq": [
            {"do": "save draft", "reads": ["text"], "changes": [{"place": "draft", "how": "create", "later": "undone"}],
             "handles": [{"when": "the note is empty", "then": "refuse"}]},
            {"do": "put on site", "reads": ["draft"], "changes": [{"place": "site", "how": "create"}],
             "handles": [{"when": "the site is down", "then": "refuse"}]},
        ]}},
        "runs": ["publish"],
        "rules": [{"rule": "a preview shows the note as it will appear", "about": "put on site", "when": "preview"}],
    }


def book():
    """An address book: import contacts from a file, each under its email."""
    return {
        "system": "contacts", "reports": "message",
        "places": [{"name": "file", "is": "one", "kind": "outside"}, {"name": "book", "is": "many", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}],
        "facts": [{"name": "the file is not a list of contacts", "fault": True, "about": "file"}],
        "parts": {"import": {"do": "import", "reads": ["file"], "changes": [{"place": "book", "how": "create", "many": True, "later": "stands", "because": "an import is not undone"}],
                             "handles": [{"when": "the file is not a list of contacts", "then": "refuse"}]}},
        "runs": ["import"],
    }


def legs():
    """A nightly run of two legs. The first stages and then publishes; the second needs nothing from the first."""
    kept = {"how": "create", "later": "stands", "because": "what landed is kept"}
    return {
        "system": "nightly", "reports": "message",
        "places": [{"name": "source", "is": "one", "kind": "outside"}, {"name": "staged", "is": "one", "kind": "lasting"}, {"name": "published", "is": "one", "kind": "lasting"},
                   {"name": "ledger", "is": "one", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}],
        "facts": [{"name": "the source is down", "fault": True, "about": "source"}],
        "parts": {"nightly": {"all": [
            {"name": "first leg", "seq": [
                {"do": "stage", "reads": ["source"], "changes": [dict(kept, place="staged")], "handles": [{"when": "the source is down", "then": "refuse"}]},
                {"do": "publish", "reads": ["staged"], "changes": [dict(kept, place="published")]}]},
            {"do": "post the ledger", "changes": [dict(kept, place="ledger")]},
        ]}},
        "runs": ["nightly"],
    }


def pipeline():
    """Read a feed, work out a total from it, then post the total. The feed can be wrong."""
    return {
        "system": "totals", "reports": "message",
        "places": [{"name": "feed", "is": "one", "kind": "outside"}, {"name": "rows", "is": "one", "kind": "run"}, {"name": "total", "is": "one", "kind": "run"},
                   {"name": "books", "is": "one", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}, {"name": "log", "is": "one", "kind": "report"}],
        "facts": [{"name": "the feed is short", "fault": True, "about": "feed"}],
        "parts": {"post": {"seq": [
            {"do": "read the feed", "reads": ["feed"], "changes": [{"place": "rows", "how": "create"}]},
            {"do": "add it up", "reads": ["rows"], "changes": [{"place": "total", "how": "create"}]},
            {"do": "post the total", "reads": ["total"], "changes": [{"place": "books", "how": "create", "later": "stands", "because": "a posted total is corrected, not removed"}]},
        ]}},
        "runs": ["post"],
    }


def store():
    """The layer below: a store that puts a thing away on whatever shelf its user gives it."""
    return {
        "system": "store", "reports": "the store's message",
        "places": [{"name": "the shelf", "is": "many", "kind": "lasting", "given": True}, {"name": "the store's message", "is": "one", "kind": "report"}],
        "facts": [{"name": "the store is closed", "fault": True, "says": "the store does not take anything now"}],
        "parts": {"put away": {"do": "put away", "changes": [{"place": "the shelf", "how": "create"}],
                               "handles": [{"when": "the store is closed", "then": "refuse"},
                                           {"when": "the shelf already has the member", "then": "accept", "because": "a second copy is shelved beside the first"}]}},
        "runs": ["put away"],
    }


def library():
    """A library that stands on the store: returning a book puts it away on the library's own shelf."""
    return {
        "system": "library", "on": ["store.json"], "reports": "message",
        "places": [{"name": "books", "is": "many", "kind": "lasting"}, {"name": "desk", "is": "one", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}],
        "facts": [{"name": "the desk is shut", "fault": True, "says": "nobody is at the desk"}],
        "parts": {"return a book": {"seq": [
            {"do": "shelve it", "uses": "put away", "with": {"the shelf": "books"}, "later": "undone",
             "handles": [{"when": "the store is closed", "then": "refuse"}, {"when": "books already has the member", "then": "refuse"}]},
            {"do": "note it at the desk", "changes": [{"place": "desk", "how": "create", "later": "stands", "because": "a note is kept"}],
             "handles": [{"when": "the desk is shut", "then": "refuse"}]},
        ]}},
        "runs": ["return a book"],
    }


def layers(top, below=None):
    """A definition written beside the one it stands on, as the engine reads them from files."""
    base = tempfile.mkdtemp()
    with open(os.path.join(base, "store.json"), "w", encoding="utf-8") as f:
        json.dump(store() if below is None else below, f)
    return engine.Design(top, base, base)


def found(d):
    return [(f["kind"], f["subject"]) for f in engine.findings(d, engine.explore_all(d))]


def one(d, *given):
    return [p for p in engine.explore_all(d) if p.given() == sorted(given)][0]


def desk():
    """Three runs over the same places: write a draft, check it, then put it on the site. Each reads what the one before left."""
    def run(do, reads, place, **more):
        return dict({"do": do, "reads": reads, "changes": [{"place": place, "how": "create", "from": reads[0], "later": "stands", "because": "kept for the next run"}]}, **more)
    return {
        "system": "desk", "reports": "message",
        "places": [{"name": "text", "is": "one", "kind": "outside"}, {"name": "draft", "is": "one", "kind": "lasting"}, {"name": "proof", "is": "one", "kind": "lasting"},
                   {"name": "site", "is": "one", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}],
        "facts": [{"name": "the note is long", "fault": True, "about": "text"}],
        "parts": {
            "write": run("write", ["text"], "draft", handles=[{"when": "the note is long", "then": "accept", "because": "a long draft can be cut later"}]),
            "check": run("check", ["draft"], "proof"),
            "publish": run("publish", ["proof"], "site"),
        },
        "runs": ["write", "check", "publish"],
    }


def play(doc, *given):
    d = engine.Design(doc)
    return [p for p in engine.explore_all(d) if p.given() == sorted(given)][0]


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
        self.assertEqual(outcomes(notes()), [((), "completed"), (("the note is empty",), "failed"), (("the site is down",), "failed")])

    def test_one_thing_goes_wrong_in_a_play_unless_the_design_allows_more(self):
        self.assertTrue(all(len(g) <= 1 for g, _ in outcomes(notes())))
        doc = notes()
        doc["faults"] = 2
        doc["parts"]["publish"]["seq"][0]["handles"][0]["then"] = "warn"      # an empty note is skipped, and the run goes on
        doc["places"][1]["kind"] = "run"
        self.assertIn((("the note is empty", "the site is down"), "failed"), outcomes(doc))

    def test_a_fact_that_makes_no_difference_makes_no_extra_play(self):
        doc = notes()
        self.assertEqual(len(outcomes(doc)), 3)                    # nothing asks for preview
        doc["parts"]["publish"]["seq"][1]["when"] = {"not": "preview"}
        self.assertEqual(len(outcomes(doc)), 4)                    # put on site asks for it, and only when it is reached

    def test_a_fact_can_be_possible_only_when_another_holds(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["when"] = {"not": "preview"}
        doc["facts"][1]["only"] = {"not": "preview"}               # an empty note is not something a preview meets
        self.assertNotIn((("preview", "the note is empty"), "failed"), outcomes(doc))
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
        for then, outcome, draft in [("refuse", "failed", "untouched"), ("warn", "completed with a warning", "untouched"),
                                     ("tell", "completed", "created"), ("accept", "completed", "created")]:
            with self.subTest(then=then):
                doc = notes()
                doc["places"][1]["kind"] = "lasting"
                doc["parts"]["publish"]["seq"][0]["handles"][0].update(then=then, because="an empty note is allowed")
                doc["parts"]["publish"]["seq"][1]["reads"] = ["text"]         # so that a skipped draft is not read
                d = engine.Design(doc)
                p = [p for p in engine.explore_all(d) if p.given() == ["the note is empty"]][0]
                self.assertEqual((p.outcome(), p.state["draft"]), (outcome, draft))
                self.assertEqual(p.state["message"], "untouched" if then == "accept" else "created")
                self.assertEqual(kinds(doc), [])                               # each is an answer

    def test_an_action_can_do_its_work_and_then_fail(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["handles"][0]["then"] = "fail"       # the site took the note and then broke
        doc["parts"]["publish"]["seq"][1]["changes"] = [{"place": "site", "how": "create"}]
        self.assertEqual(kinds(doc), [("left-behind", "site")])                 # its own change needs an answer too
        doc["parts"]["publish"]["seq"][1]["changes"][0]["later"] = "undone"
        self.assertEqual(kinds(doc), [])
        d = engine.Design(doc)
        p = [p for p in engine.explore_all(d) if p.given() == ["the site is down"]][0]
        self.assertEqual((p.outcome(), p.state["site"], p.state["draft"], p.steps[-1]), ("failed", "undone", "undone", ("put on site", "failed")))

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
        self.assertIn((("the note is empty",), "failed"), outcomes(doc))
        self.assertNotIn((("preview", "the note is empty"), "completed"), outcomes(doc))

    def test_a_bad_value_can_be_accepted_under_a_fact(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["handles"] = [{"when": {"all": ["the note is empty", {"not": "preview"}]}, "then": "refuse"}]
        self.assertEqual(kinds(doc), [("unanswered", "the note is empty")])    # in a preview nothing answers it
        doc["parts"]["publish"]["seq"][0]["handles"].append({"when": {"all": ["the note is empty", "preview"]}, "then": "accept", "because": "a preview shows whatever is there"})
        self.assertEqual(kinds(doc), [])

    def test_asking_a_person_plays_both_answers(self):
        doc = book()
        self.assertEqual(kinds(doc), [("unanswered", "book already has the member"), ("unanswered", "book is given a key twice")])
        doc["parts"]["import"]["handles"] += [{"when": "book already has the member", "then": "ask"}, {"when": "book is given a key twice", "then": "refuse"}]
        self.assertEqual(kinds(doc), [])
        self.assertEqual(outcomes(doc), [
            ((), "completed"), (("book already has the member",), "declined"),
            (("book already has the member", "the person says yes at import"), "completed"),
            (("book is given a key twice",), "failed"), (("the file is not a list of contacts",), "failed")])
        doc["parts"]["import"]["handles"][1]["no"] = "skip"
        self.assertIn((("book already has the member",), "completed"), outcomes(doc))

    def test_an_answer_with_no_when_always_applies(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][1]["handles"].append({"then": "ask"})      # the person is always asked before the site is changed
        self.assertEqual(kinds(doc), [])
        got = outcomes(doc)
        self.assertIn(((), "declined"), got)
        self.assertIn((("the person says yes at put on site",), "completed"), got)
        self.assertIn("put on site: ask, always", [x["subject"] for x in engine.tests(engine.Design(doc))])

    def test_an_existing_member_can_be_normal_for_a_place(self):
        doc = book()
        doc["parts"]["import"]["handles"].append({"when": "book is given a key twice", "then": "refuse"})
        self.assertEqual(kinds(doc), [("unanswered", "book already has the member")])
        doc["facts"].append({"name": "book already has the member", "about": "book", "says": "an import is run again over the same book"})
        doc["rules"] = [{"rule": "a contact already in the book is brought up to date", "about": "import", "when": "book already has the member"}]
        self.assertEqual(kinds(doc), [])                                  # a state of the world now, and it no longer uses up the one fault
        self.assertIn((("book already has the member", "the file is not a list of contacts"), "failed"), outcomes(doc))

    def test_replacing_a_member_asks_whose_it_is(self):
        doc = book()
        doc["parts"]["import"]["changes"] = [{"place": "book", "how": "replace", "later": "stands", "because": "an import is not undone"}]
        self.assertEqual(kinds(doc), [("unanswered", "book holds another's member"), ("unanswered", "book holds what is kept nowhere")])
        doc["places"][1]["is"] = "one"                     # a place that holds one state has no members to be another's
        self.assertEqual(kinds(doc), [("unanswered", "book holds what is kept nowhere")])

    def test_a_change_says_what_happens_to_it_when_the_run_stops_later(self):
        doc = notes()
        del doc["parts"]["publish"]["seq"][0]["changes"][0]["later"]
        self.assertEqual(kinds(doc), [("left-behind", "draft")])
        for later, state in [("undone", "undone"), ("stands", "created")]:
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
            {"name": "the site is written after the draft", "before": "put on site", "place": "draft", "is": "created"},
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
        doc["parts"]["publish twice"] = {"seq": ["publish", {"do": "say done", "changes": [{"place": "message", "how": "create"}]}]}
        doc["runs"].append("publish twice")
        doc["facts"][1]["only"] = "in publish"                                # an empty note matters to publish alone
        self.assertEqual([o for o in outcomes(doc) if "the note is empty" in o[0]], [(("the note is empty",), "failed")])
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
        doc["parts"]["publish"]["all"][1]["changes"] = [{"place": "site", "how": "create", "later": "stands", "because": "what is on the site stays"}]   # in no order, it runs though the draft was refused
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
        doc["places"][2] = {"name": "site", "is": "one", "kind": "outside"}
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
        doc["parts"]["publish twice"] = {"seq": ["publish", {"do": "say done", "changes": [{"place": "message", "how": "create"}]}]}
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
        self.assertEqual([f["subject"] for f in got], ["draft ends undone in the design and created in the code"])
        self.assertIn("Every play where this happens has: the site is down.", got[0]["shows"])
        code = notes()
        code["facts"].append({"name": "the disk is full", "fault": True})   # the code refuses for a reason the design never names
        code["parts"]["publish"]["seq"][0]["handles"].append({"when": "the disk is full", "then": "refuse"})
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], ["save draft: the design goes ahead, the code refuses"])
        self.assertIn("Every play where this happens has: the disk is full.", got[0]["shows"])

    def test_a_step_that_tells_differs_from_one_that_goes_on_without_a_word(self):
        design, code = notes(), notes()
        design["facts"].append({"name": "the note is long", "about": "text"})
        code["facts"].append({"name": "the note is long", "about": "text"})
        design["parts"]["publish"]["seq"][0]["handles"].append({"when": "the note is long", "then": "tell"})
        code["parts"]["publish"]["seq"][0]["handles"].append({"when": "the note is long", "then": "accept", "because": "a long note is fine"})
        design["parts"]["publish"]["seq"][1]["handles"].append({"when": "preview", "then": "accept", "because": "a preview is shown"})
        code["parts"]["publish"]["seq"][1]["handles"].append({"when": "preview", "then": "tell"})
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], [                                       # one difference at each step, named for the step
            "put on site: the design goes ahead, the code tells and goes on",
            "save draft: the design tells and goes on, the code goes ahead"])
        self.assertIn("Every play where this happens has: the note is long.", got[1]["shows"])

    def test_an_action_the_design_skips_and_the_code_lacks_is_no_difference(self):
        design, code = notes(), notes()
        design["parts"]["publish"]["seq"].insert(0, {"do": "spell check", "when": "preview", "reads": ["text"]})
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([f["subject"] for f in got], ["spell check: the design goes ahead, the code has no such action"])   # where it runs, the code lacks it
        self.assertIn("Every play where this happens has: preview.", got[0]["shows"])      # where it is skipped, both do nothing

    def test_an_action_skipped_in_one_place_and_run_in_another_is_no_difference(self):
        design = notes()
        design["parts"]["check"] = {"do": "check", "reads": ["text"]}
        design["parts"]["publish"]["seq"][1:1] = ["check"]
        design["parts"]["publish"]["seq"].insert(0, {"use": "check", "when": "preview"})   # checked first in a preview, and always after the draft
        self.assertEqual(engine.compare(engine.Design(design), engine.Design(design)), [])
        code = notes()
        code["parts"]["check"] = {"do": "check", "reads": ["text"]}
        code["parts"]["publish"]["seq"][1:1] = ["check"]                                    # the code checks once, after the draft
        got = engine.compare(engine.Design(design), engine.Design(code))
        self.assertEqual([(f["subject"], f["example"]["given"]) for f in got],                 # where the first check is skipped, both check once
                         [("check: the design goes ahead, the code has no such action", ["preview", "the note is empty"])])   # a preview of an empty note: only the design checks before the draft refuses

    def test_a_failure_ends_a_sequence_and_the_parts_beside_it_still_run(self):
        p = play(legs(), "the source is down")
        self.assertEqual((p.outcome(), p.stop.action), ("failed", "stage"))
        self.assertEqual(p.steps, [("stage", "refused"), ("post the ledger", "ran")])      # publish needs what stage produced; the ledger does not
        doc = legs()
        doc["parts"]["nightly"] = {"seq": doc["parts"]["nightly"]["all"]}                   # in a sequence the ledger waits on the first leg
        self.assertEqual(play(doc, "the source is down").steps, [("stage", "refused")])

    def test_a_part_can_answer_a_failure_its_children_return(self):
        doc = legs()
        doc["parts"]["nightly"]["all"][0]["handles"] = [{"when": "the source is down", "then": "warn"}]
        p = play(doc, "the source is down")
        self.assertEqual((p.outcome(), p.stop), ("completed with a warning", None))
        self.assertEqual(p.steps, [("stage", "refused"), ("post the ledger", "ran")])      # the sequence still ended at the failure
        self.assertEqual(p.state["message"], "created")                                    # and the person is told
        doc["parts"]["nightly"]["all"][0]["handles"] = [{"when": "the source is down", "then": "accept"}]
        self.assertEqual(kinds(doc), [("unexplained", "first leg: accept, when the source is down")])
        doc["parts"]["nightly"]["all"][0]["handles"] = [{"when": "the source is down", "then": "refuse"}]
        with self.assertRaises(engine.Refused) as cm:
            engine.Design(doc)
        self.assertIn("warn, tell or accept", str(cm.exception))

    def test_what_happens_later_is_decided_as_a_failure_leaves_each_part(self):
        doc = legs()
        first = doc["parts"]["nightly"]["all"][0]
        first["seq"] = [{"do": "open the ledger", "changes": [{"place": "ledger", "how": "create"}]},
                        {"name": "staging", "seq": [{"do": "mark", "changes": [{"place": "published", "how": "create", "later": "undone"}]}, first["seq"][0]]}]
        doc["parts"]["nightly"] = first
        self.assertEqual(kinds(doc), [("left-behind", "ledger")])                         # the failure leaves the whole run, and the ledger was opened in it
        first["handles"] = [{"when": "the source is down", "then": "warn"}]
        self.assertEqual(kinds(doc), [])                                                   # answered at the run's own part, the failure never leaves it
        p = play(doc, "the source is down")
        self.assertEqual((p.state["published"], p.state["ledger"]), ("undone", "created"))     # it did leave the part that marked, so the mark is undone

    def test_a_fault_is_asked_where_something_lasting_is_changed_on_the_strength_of_it(self):
        d = engine.Design(pipeline())
        found = [f for f in engine.findings(d, engine.explore_all(d)) if f["kind"] == "unanswered"]
        self.assertEqual([(f["subject"], f["actions"], f["rest"]) for f in found], [("the feed is short", ["post the total"], ["rows", "total", "books"])])
        doc = pipeline()                                   # a later action can answer for an earlier read, before anything lasting changes
        doc["parts"]["post"]["seq"][1]["handles"] = [{"when": "the feed is short", "then": "refuse"}]
        self.assertEqual(kinds(doc), [])
        doc = pipeline()                                   # answered after the lasting change is too late
        doc["parts"]["post"]["seq"].append({"do": "check the feed", "reads": ["feed"], "handles": [{"when": "the feed is short", "then": "refuse"}]})
        self.assertEqual(kinds(doc), [("unanswered", "the feed is short")])
        doc = pipeline()                                   # a value that never reaches anything lasting is no question
        del doc["parts"]["post"]["seq"][2]
        self.assertEqual(kinds(doc), [])

    def test_a_fault_can_say_the_step_it_goes_wrong_at(self):
        doc = pipeline()
        doc["facts"].append({"name": "the sum overflows", "fault": True, "at": "add it up"})
        self.assertEqual(kinds(doc), [("unanswered", "the feed is short"), ("unanswered", "the sum overflows")])
        d = engine.Design(doc)
        at = [f["actions"] for f in engine.findings(d, engine.explore_all(d)) if f["subject"] == "the sum overflows"]
        self.assertEqual(at, [["add it up"]])              # asked at its step, whatever the step changes
        doc["parts"]["post"]["seq"][1]["handles"] = [{"when": "the sum overflows", "then": "refuse"}]
        self.assertEqual(kinds(doc), [("unanswered", "the feed is short")])
        doc["facts"][-1]["at"] = ["add it up", "balance"]
        with self.assertRaises(engine.Refused) as cm:
            engine.Design(doc)
        self.assertIn("\"balance\", which is not an action", str(cm.exception))

    def test_an_answer_can_tell_more_than_one_place(self):
        doc = pipeline()
        doc["parts"]["post"]["seq"][0]["handles"] = [{"when": "the feed is short", "then": "refuse", "tells": ["message", "log"]}]
        p = play(doc, "the feed is short")
        self.assertEqual((p.state["message"], p.state["log"]), ("created", "created"))
        doc["parts"]["post"]["seq"][0]["handles"][0]["tells"] = ["message", "ledger"]
        with self.assertRaises(engine.Refused):
            engine.Design(doc)

    def test_a_use_can_carry_the_sentence_it_stands_for(self):
        doc = pipeline()
        doc["parts"]["nightly"] = {"seq": [{"use": "post", "says": "The totals are posted each night."}]}
        doc["runs"] = ["nightly"]
        d = engine.Design(doc)
        self.assertIn(("The totals are posted each night.", "played"), engine.said(d))

    def test_replacing_what_lasts_asks_what_happens_to_what_was_there(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["changes"][0]["how"] = "replace"                 # saving over the last draft loses it
        self.assertEqual(kinds(doc), [("unanswered", "draft holds what is kept nowhere")])
        doc["parts"]["publish"]["seq"][0]["handles"].append({"when": "draft holds what is kept nowhere", "then": "refuse"})
        self.assertEqual(kinds(doc), [])
        doc["places"][1]["kind"] = "run"                   # what lasts for one run is not asked about
        del doc["parts"]["publish"]["seq"][0]["handles"][1]
        self.assertEqual(kinds(doc), [])

    def test_a_place_copied_to_another_is_kept_until_it_changes_again(self):
        doc = notes()
        seq = doc["parts"]["publish"]["seq"]
        seq[0]["changes"][0]["how"] = "replace"
        seq[0]["handles"].append({"when": "draft holds what is kept nowhere", "then": "refuse"})
        seq[1]["changes"] = [{"place": "site", "how": "create", "from": "draft"}]           # the site now holds a copy of the draft
        seq.append({"do": "clear draft", "changes": [{"place": "draft", "how": "replace"}]})
        self.assertEqual(kinds(doc), [])                                                    # so clearing the draft loses nothing
        del seq[1]["changes"][0]["from"]
        self.assertEqual(kinds(doc), [("unanswered", "draft holds what is kept nowhere")])  # with no copy it does, in every play

    def test_destroying_asks_who_else_holds_it(self):
        doc = notes()
        gone = {"place": "draft", "how": "destroy", "later": "stands", "because": "the old draft is not wanted back"}
        doc["parts"]["publish"]["seq"].insert(0, {"do": "discard the old draft", "changes": [gone]})
        self.assertEqual(kinds(doc), [("unanswered", "draft is held elsewhere")])
        doc["parts"]["publish"]["seq"][0]["handles"] = [{"when": "draft is held elsewhere", "then": "refuse"}]
        self.assertEqual(kinds(doc), [])
        doc = notes()                                      # a draft this run made is held nowhere else, until the site takes a copy
        seq = doc["parts"]["publish"]["seq"]
        seq.append({"do": "discard draft", "changes": [gone], "handles": [{"when": "draft is held elsewhere", "then": "refuse"}]})
        self.assertEqual(play(doc).steps[-1], ("discard draft", "ran"))
        seq[1]["changes"] = [{"place": "site", "how": "create", "from": "draft"}]
        self.assertEqual(play(doc).steps[-1], ("discard draft", "refused"))

    def test_a_create_leaves_what_was_there_and_a_replace_ends_it(self):
        def tags(how):
            return {
                "system": "versions", "reports": "message",
                "places": [{"name": "remote", "is": "many", "kind": "outside"}, {"name": "tags", "is": "many", "kind": "lasting"},
                           {"name": "version", "is": "one", "kind": "lasting"}, {"name": "message", "is": "one", "kind": "report"}],
                "facts": [{"name": "a tag does not name a version", "fault": True, "about": "tags"},
                          {"name": "tags already has the member", "about": "tags", "says": "a tag is fetched that is here already"}],
                "parts": {"next": {"seq": [
                    {"do": "fetch", "reads": ["remote"], "changes": [{"place": "tags", "how": how, "later": "stands", "because": "fetched tags stay"}],
                     "handles": [{"when": {"any": ["tags holds what is kept nowhere", "tags holds another's member"]}, "then": "accept", "because": "the remote has them"}]},
                    {"do": "work out the version", "reads": ["tags"], "changes": [{"place": "version", "how": "create"}]}]}},
                "runs": ["next"],
            }
        self.assertEqual(kinds(tags("create")), [("unanswered", "a tag does not name a version")])   # the tags that were there are still there
        self.assertEqual(kinds(tags("replace")), [("unused", "a tag does not name a version"), ("unused", "tags already has the member")])

    def test_what_comes_from_a_place_carries_what_is_wrong_with_it(self):
        doc = notes()
        doc["facts"].append({"name": "the old draft is damaged", "fault": True, "about": "draft"})
        doc["parts"]["publish"]["seq"] = [{"do": "put on site", "changes": [{"place": "site", "how": "create", "from": "draft"}],
                                           "handles": [{"when": "the site is down", "then": "refuse"}]}]
        self.assertEqual(kinds(doc), [("unanswered", "the old draft is damaged"), ("unused", "the note is empty")])   # asked though the action lists no read

    def test_a_place_can_be_in_another(self):
        doc = notes()
        doc["places"].insert(1, {"name": "folder", "kind": "lasting"})                      # made of the places in it
        doc["places"][2]["in"] = "folder"
        doc["expects"] = [{"name": "a preview leaves the folder alone", "when": "preview", "never changes": ["folder"]}]
        self.assertEqual(kinds(doc), [("expect", "a preview leaves the folder alone")])     # what is done to the draft is done to the folder
        doc["expects"] = []
        seq = doc["parts"]["publish"]["seq"]
        seq[0]["changes"][0]["how"] = "replace"
        self.assertEqual(kinds(doc), [("unanswered", "draft holds what is kept nowhere")])
        seq.insert(0, {"do": "check the folder", "reads": ["folder"], "handles": [{"when": "folder holds what is kept nowhere", "then": "refuse"}]})
        self.assertEqual(kinds(doc), [])                                                    # an answer about the folder answers for the draft in it

    def test_the_page_lists_what_a_place_brings_as_the_engine_has_it(self):
        with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "DESIGN.md"), encoding="utf-8") as f:
            page = f.read()
        table = page[page.index("<!-- brought"):page.index("<!-- /brought -->")]
        self.assertEqual(sorted(re.findall(r"`(<place> [^`]+)`", table)), sorted(name % "<place>" for name, _ in engine.BROUGHT.values()))

    def test_an_action_can_use_a_run_of_the_definition_below(self):
        d = layers(library())
        self.assertEqual(found(d), [])
        p = one(d)
        self.assertEqual((p.steps, p.state["books"], p.state["the shelf"]), ([("shelve it", "ran"), ("note it at the desk", "ran")], "created", "untouched"))
        self.assertEqual((one(d, "the store is closed").steps, one(d, "books already has the member").steps), ([("shelve it", "refused")], [("shelve it", "refused")]))

    def test_a_failure_from_below_that_nothing_answers_is_a_question(self):
        doc = library()
        del doc["parts"]["return a book"]["seq"][0]["handles"][0]
        d = layers(doc)
        self.assertEqual(found(d), [("unanswered", "the store is closed")])
        p = one(d, "the store is closed")
        self.assertEqual((p.outcome(), p.stop.action, p.steps), ("failed", "shelve it", [("shelve it", "failed")]))   # the run below refused, and it fails here

    def test_a_failure_from_below_told_of_lets_the_run_go_on(self):
        doc = library()
        doc["parts"]["return a book"]["seq"][0]["handles"][0] = {"when": "the store is closed", "then": "tell"}
        d = layers(doc)
        self.assertEqual(found(d), [])
        p = one(d, "the store is closed")                  # the person is told, and the run goes on
        self.assertEqual((p.outcome(), p.steps, p.state["message"]), ("completed", [("shelve it", "ran"), ("note it at the desk", "ran")], "created"))
        doc["parts"]["return a book"]["seq"][0]["handles"][0] = {"when": "the store is closed", "then": "accept", "because": "a closed store is tried again tomorrow"}
        p = one(layers(doc), "the store is closed")        # accepted, it goes on and the person is not told
        self.assertEqual((p.outcome(), p.steps, p.state["message"]), ("completed", [("shelve it", "ran"), ("note it at the desk", "ran")], "untouched"))

    def test_what_the_run_below_accepted_is_asked_again(self):
        doc = library()
        del doc["parts"]["return a book"]["seq"][0]["handles"][1]
        d = layers(doc)
        self.assertEqual(found(d), [("unanswered", "books already has the member")])       # named for the place given, and the store's reason is not this one's
        doc["parts"]["return a book"]["seq"][0]["handles"].append({"when": "books already has the member", "then": "accept", "because": "the library keeps two copies"})
        self.assertEqual(found(layers(doc)), [])

    def test_a_rehearsal_makes_no_changes_and_every_answer_still_applies(self):
        doc = library()
        doc["facts"] += [{"name": "dry run", "says": "--dry-run is given"}, {"name": "a book is back", "about": "books"}]
        doc["parts"]["return a book"]["seq"][0].update(rehearsal="dry run", makes={"a book is back": True})
        doc["parts"]["return a book"]["seq"][1]["when"] = "a book is back"
        d = layers(doc)
        p = one(d, "dry run")
        self.assertEqual((p.state["books"], p.steps[0], p.now.made.get("a book is back")), ("untouched", ("shelve it", "ran"), None))
        self.assertEqual(one(d).state["books"], "created")
        del doc["parts"]["return a book"]["seq"][0]["handles"][0]                          # the store's own refusal still applies in a rehearsal
        self.assertEqual(one(layers(doc), "dry run", "the store is closed").steps, [("shelve it", "failed")])

    def test_an_action_that_uses_a_run_says_what_it_makes_and_what_happens_later(self):
        doc = library()
        doc["facts"].append({"name": "a book is back", "about": "books"})
        doc["parts"]["return a book"]["seq"][0]["makes"] = {"a book is back": True}
        doc["rules"] = [{"rule": "a returned book can be lent again", "about": "note it at the desk", "when": "a book is back"}]
        d = layers(doc)
        self.assertEqual(one(d).now["a book is back"], True)
        self.assertEqual(one(d, "the desk is shut").state["books"], "undone")            # the failure leaves the sequence, and the shelving is undone
        doc = library()
        del doc["parts"]["return a book"]["seq"][0]["later"]
        self.assertEqual(found(layers(doc)), [("left-behind", "books")])
        doc["parts"]["return a book"]["seq"][0]["later"] = "stands"
        self.assertEqual(found(layers(doc)), [("unexplained", "shelve it: what it changed stands")])

    def test_a_given_place_is_supplied_by_whoever_uses_the_run(self):
        doc = store()
        doc["places"].append({"name": "the slip", "is": "one", "kind": "run", "given": True})
        doc["parts"]["put away"]["reads"] = ["the slip"]
        self.assertEqual(kinds(doc), [])                   # played alone, a given place is not one the run forgot to write

    def test_a_fault_can_go_wrong_at_an_action_that_uses_a_run(self):
        doc = library()
        doc["facts"].append({"name": "the book is the wrong one", "fault": True, "at": "shelve it", "says": "the store shelves what it is handed"})
        self.assertEqual(found(layers(doc)), [("unanswered", "the book is the wrong one")])   # the store knows nothing of it, so it is asked here
        doc["parts"]["return a book"]["seq"][0]["handles"].append({"when": "the book is the wrong one", "then": "refuse"})
        self.assertEqual(found(layers(doc)), [])

    def test_a_definition_that_misuses_the_one_below_is_refused_with_the_reason(self):
        cases = [
            (lambda d: d["parts"]["return a book"]["seq"][0].update(uses="burn"), '"burn", which is not a run'),
            (lambda d: d["parts"]["return a book"]["seq"][0].update(reads=["desk"]), "states no reads and no changes"),
            (lambda d: d["parts"]["return a book"]["seq"][0].update({"with": {}}), 'which place is "the shelf"'),
            (lambda d: d["parts"]["return a book"]["seq"][0].update({"with": {"the shelf": "desk"}}), "holds many"),
            (lambda d: d["parts"]["return a book"]["seq"][0].update({"with": {"the shelf": "books", "the store's message": "message"}}), "not a given place"),
            (lambda d: d["places"].append({"name": "the shelf", "is": "many", "kind": "lasting"}), 'two places are named "the shelf"'),
            (lambda d: d["facts"].append({"name": "the store is closed", "fault": True}), 'two facts are named "the store is closed"'),
            (lambda d: d.update(on=["cellar.json"]), '"cellar.json"'),
        ]
        layers(library())
        for change, reason in cases:
            with self.subTest(reason=reason):
                doc = library()
                change(doc)
                with self.assertRaises(engine.Refused) as cm:
                    layers(doc)
                self.assertIn(reason, str(cm.exception))
        with self.assertRaises(engine.Refused) as cm:
            engine.Design(library())                       # with no directory given, the definition below cannot be found
        self.assertIn("directory", str(cm.exception))

    def test_what_one_run_accepted_is_asked_again_of_the_run_that_reads_what_it_left(self):
        d = engine.Design(desk())
        fs = [f for f in engine.findings(d, engine.explore_all(d))]
        self.assertEqual([(f["kind"], f["subject"], f["actions"]) for f in fs], [("unanswered", "the note is long", ["check"])])
        self.assertIn("write can leave draft made with it", " ".join(fs[0]["shows"]))
        doc = desk()                                       # the run that reads it gives its own answer
        doc["parts"]["check"]["handles"] = [{"when": "the note is long", "then": "refuse"}]
        self.assertEqual(kinds(doc), [])
        doc = desk()                                       # refused where it arises, it is left in nothing
        doc["parts"]["write"]["handles"][0] = {"when": "the note is long", "then": "refuse"}
        self.assertEqual(kinds(doc), [])

    def test_it_reaches_a_third_run_through_the_second(self):
        doc = desk()
        doc["parts"]["check"]["handles"] = [{"when": "the note is long", "then": "accept", "because": "the proof is marked long"}]
        d = engine.Design(doc)
        fs = engine.findings(d, engine.explore_all(d))
        self.assertEqual([(f["kind"], f["subject"], f["actions"]) for f in fs], [("unanswered", "the note is long", ["publish"])])

    def test_a_fact_no_run_makes_true_is_false_when_a_run_starts(self):
        doc = notes()
        doc["facts"].append({"name": "a draft is waiting", "about": "draft"})
        seq = doc["parts"]["publish"]["seq"]
        seq[1]["changes"][0]["makes"] = {"a draft is waiting": False}
        seq.insert(0, {"do": "remind", "when": "a draft is waiting", "changes": [{"place": "message", "how": "create"}]})
        self.assertEqual([p for p in engine.explore_all(engine.Design(doc)) if p.ran("remind")], [])   # nothing in the model leaves a draft waiting
        seq[1]["changes"][0]["makes"] = {"a draft is waiting": True}
        self.assertTrue([p for p in engine.explore_all(engine.Design(doc)) if p.ran("remind")])         # now an earlier run can have left one

    def test_the_page_lists_every_key_the_engine_accepts(self):
        """One way only: a key the engine accepts and the page does not list. "says" and "hold" are stated once, under the table."""
        rows = {"The design": "design", "A place": "place", "A fact": "fact", "An action": "action", "A read": "read", "A change": "change", "An answer": "handle",
                "A sequence": "seq", "Parts that do not depend on each other": "all", "A choice": "choose", "A use of a part": "use",
                "An expectation": "expect", "A rule in words": "rule", "A hold": "hold", "An acceptances file": "accepted", "An acceptance": "acceptance",
                "A branch of a choice": "branch"}
        with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "DESIGN.md"), encoding="utf-8") as f:
            page = f.read()
        table = page[page.index("<!-- keys"):page.index("<!-- /keys -->")]
        found = {}
        for line in table.splitlines():
            cells = [c.strip() for c in line.strip().strip("|").split("|")]
            if len(cells) == 2 and cells[0] in rows:
                found[rows[cells[0]]] = set(re.findall(r"`([a-z ]+)`", cells[1])) | set(re.findall(r'"([a-z ]+)"', cells[1]))
        self.assertEqual(sorted(found), sorted(engine.KEYS))
        missing = {kind: sorted(keys - found[kind] - {"says", "hold"}) for kind, keys in engine.KEYS.items()}
        self.assertEqual({k: v for k, v in missing.items() if v}, {})

    def test_a_statement_is_printed_with_the_whole_of_its_condition(self):
        doc = notes()
        doc["parts"]["publish"]["seq"][0]["handles"] = [{"when": {"not": "preview"}, "then": "tell"},
                                                        {"when": {"all": ["preview", {"any": ["the note is empty", "the site is down"]}]}, "then": "refuse"}]
        subjects = [t["subject"] for t in engine.tests(engine.Design(doc))]
        self.assertIn("save draft: tell, when not preview", subjects)                      # a "not" is part of what the answer says
        self.assertIn("save draft: refuse, when preview and (the note is empty or the site is down)", subjects)

    def test_a_sentence_that_ends_in_a_full_stop_is_printed_with_one(self):
        doc = notes()
        doc["facts"][1]["says"] = "The note has no text."
        del doc["parts"]["publish"]["seq"][0]["handles"]
        d = engine.Design(doc)
        shows = engine.findings(d, engine.explore_all(d))[0]["shows"]
        self.assertEqual(shows[0], "The note has no text.")

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
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=[{"place": "site"}]), "says how"),
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=["site"]), "says how"),
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=[{"place": "site", "how": "set"}]), "create, replace, destroy"),
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=[{"place": "site", "how": "create", "many": True}]), "holds many"),
            (lambda d: d["parts"]["publish"]["seq"][1].update(changes=[{"place": "site", "how": "create", "from": "nowhere"}]), '"nowhere"'),
            (lambda d: d["places"][1].pop("is"), "made of the places in it"),
            (lambda d: d["places"][1].update({"is": "pile"}), "one, many, series"),
            (lambda d: d["places"][1].update({"in": "nowhere"}), '"nowhere"'),
            (lambda d: (d["places"][1].update({"in": "site"}), d["places"][2].update({"in": "draft"})), "is in itself"),
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


def named(written=False):
    """notes(), where putting it on the site is for a note that is not a preview: "publishing", named once, or written out where it is used."""
    doc = notes()
    publishing, live = {"not": "preview"}, {"all": ["publishing", {"not": "the site is down"}]}
    if written:
        publishing, live = publishing, {"all": [publishing, {"not": "the site is down"}]}
    else:
        doc["facts"] += [{"name": "publishing", "means": publishing, "says": "the note is going to the site"},
                         {"name": "live", "means": live}]
    put = doc["parts"]["publish"]["seq"][1]
    put["when"] = publishing if written else "publishing"
    put["handles"][0]["when"] = {"all": [publishing if written else "publishing", "the site is down"]}
    doc["rules"][0]["when"] = live if written else "live"
    return doc


def everything(doc):
    """Every play as the engine plays it, and what it finds."""
    d = engine.Design(doc)
    plays = engine.explore_all(d)
    return (sorted((p.run, tuple(p.given()), p.outcome(), tuple(p.steps), tuple(sorted(p.state.items()))) for p in plays),
            engine.findings(d, plays))


class TextSyntax(unittest.TestCase):
    def test_the_text_syntax_fixture_uses_every_key_the_engine_reads(self):
        """lab/syntax round-trips this design through its text forms, so a key the engine
        gains fails here until the fixture uses it, and there until the syntax prints it."""
        path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "syntax", "testdata", "every-key.json")
        seen, was = {}, engine.only
        def recording(obj, kind, where):
            if isinstance(obj, dict):
                seen.setdefault(kind, set()).update(obj)
            return was(obj, kind, where)
        engine.only = recording
        try:
            with open(path, encoding="utf-8") as f:
                doc = json.load(f)
            engine.Design(doc, ".", os.path.dirname(path))
        finally:
            engine.only = was
        def holds(x):                                            # a hold is read only when its test is looked up, so its keys are taken from the file
            if isinstance(x, dict):
                if isinstance(x.get("hold"), dict):
                    seen.setdefault("hold", set()).update(x["hold"])
                for v in x.values():
                    holds(v)
            elif isinstance(x, list):
                for v in x:
                    holds(v)
        holds(doc)
        files = {"accepted", "acceptance"}                       # an acceptances file is not a design, and has no text form yet
        self.assertEqual({k: sorted(v - seen.get(k, set())) for k, v in engine.KEYS.items() if k not in files and v - seen.get(k, set())}, {})


class Named(unittest.TestCase):
    def test_a_design_with_a_named_fact_plays_as_the_one_with_its_meaning_written_out(self):
        self.assertEqual(everything(named()), everything(named(written=True)))
        d = engine.Design(named())
        self.assertFalse(any("publishing" in p.env or "live" in p.env for p in engine.explore_all(d)))   # a named fact is never played
        self.assertNotIn("publishing", d.facts)
        doc = named()
        doc["rules"][0]["when"] = "preview"                                            # nothing names "live" now
        self.assertEqual([(f["kind"], f["subject"]) for f in engine.findings(engine.Design(doc), engine.explore_all(engine.Design(doc)))], [("unused", "live")])

    def test_an_answer_is_put_in_words_with_the_name(self):
        subjects = [t["subject"] for t in engine.tests(engine.Design(named()))]
        self.assertIn("put on site: refuse, when publishing and the site is down", subjects)

    def test_the_callers_document_is_left_as_it_was(self):
        doc = named()
        was = copy.deepcopy(doc)
        engine.Design(doc)
        self.assertEqual(doc, was)

    def test_a_description_of_code_can_give_a_name_a_meaning_of_its_own(self):
        code = named()
        code["facts"][-2]["means"] = {"not": "the site is down"}                     # the code puts a preview on the site too
        got = engine.compare(engine.Design(named()), engine.Design(code))
        self.assertEqual([(f["subject"], f["example"]["given"]) for f in got],                  # the facts are compared, and the name is not
                         [("put on site: the design refuses, the code skips it", ["the site is down"]),
                          ("put on site: the design skips it, the code goes ahead", ["preview"])])

    def test_what_a_named_fact_cannot_be(self):
        def refused(change, reason):
            doc = named()
            change(doc)
            with self.assertRaises(engine.Refused) as cm:
                engine.Design(doc)
            self.assertIn(reason, str(cm.exception))
        for key, value in (("fault", True), ("open", True), ("about", "text"), ("at", "save draft"), ("only", "preview")):
            with self.subTest(key=key):
                refused(lambda doc: doc["facts"][-2].update({key: value}),
                        "the named fact \"publishing\" has \"%s\": a named fact holds when its meaning does" % key)
        cases = [
            (lambda doc: doc["facts"][-2].update(means={"all": ["live", "preview"]}), "the named fact \"publishing\" leads back to itself: publishing, live, publishing"),
            (lambda doc: doc["facts"][-2].update(means="the note is late"), "the named fact \"publishing\" names the fact \"the note is late\", which the design does not list"),
            (lambda doc: doc["facts"][-2].update(means={"not": "preview", "all": []}), "where a fact is asked for, give a name"),
            (lambda doc: doc["parts"]["publish"]["seq"][1].update(when={"not": "preview", "all": []}), "where a fact is asked for, give a name"),
            (lambda doc: doc["facts"].append({"name": "preview", "means": "the site is down"}), "two facts are named \"preview\""),
            (lambda doc: doc["parts"]["publish"]["seq"][0].update(makes={"publishing": True}), "makes the named fact \"publishing\" true or false: a named fact holds when its meaning does"),
            (lambda doc: doc["parts"]["publish"]["seq"][0].update(when={"named": ["publishing", "preview"]}), "{\"named\": ...}"),
        ]
        for change, reason in cases:
            with self.subTest(reason=reason):
                refused(change, reason)


if __name__ == "__main__":
    unittest.main()
