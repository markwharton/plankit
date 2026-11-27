"""What DESIGN.md says the engine finds on pk, held to the files.

    python3 -m unittest test_pk     (from this directory)
"""
import collections
import json
import os
import unittest

import engine

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.normpath(os.path.join(HERE, "..", ".."))


def load(name):
    with open(os.path.join(HERE, name), encoding="utf-8") as f:
        return engine.Design(json.load(f), REPO, HERE)


class Pk(unittest.TestCase):
    def test_sixteen_of_the_21_statements_are_played_and_five_stay_words(self):
        with open(os.path.join(REPO, "definition", "pk.json"), encoding="utf-8") as f:
            words = [u["what"] for u in json.load(f)["unsaid"]]
        stated = {}
        for s, how in engine.said(load("pk.json")):
            if s in words:
                stated[s] = "played" if how == "played" or stated.get(s) == "played" else "words"
        self.assertEqual(len(words), 21)
        self.assertEqual(collections.Counter(stated.values()), {"played": 16, "words": 5})
        self.assertEqual([w for w in words if w not in stated], [])

    def test_the_plays_and_the_questions_are_the_ones_the_page_gives(self):
        d = load("pk.json")
        plays = engine.explore_all(d)
        self.assertEqual(collections.Counter(p.run for p in plays), {"changelog": 90, "release": 44, "ship": 261})
        found = [(f["kind"], f["subject"]) for f in engine.findings(d, plays)]
        self.assertEqual(found, [
            ("open", "a rejected tag update fails the fetch"),                 # git's manual does not say, and pk's fetch depends on it
            ("expect", "changelog with --dry-run writes nothing"),
            ("unanswered", "a tag does not name a version"),
            ("unanswered", "origin's tags already has the member"),            # comes up from git's push, and no push action answers it
            ("unanswered", "settings are invalid"),
            ("unanswered", "the branch is not the default"),
            ("unanswered", "the working branch is unfit"),
            ("left-behind", "remote-tracking branch"),
            ("left-behind", "tags")])

    def test_what_git_goes_on_with_pk_refuses(self):
        d = load("pk.json")
        plays = engine.explore(d, "changelog")
        for fault in ("working branch is held elsewhere", "working tree holds what is kept nowhere"):   # git's reset accepts both
            mine = [p for p in plays if p.env.get("undo") and p.env.get(fault)]
            self.assertTrue(mine, fault)
            self.assertEqual({(p.outcome(), tuple(p.steps)) for p in mine}, {("failed", (("undo", "refused"),))}, fault)

    def test_a_bad_tag_is_asked_about_where_the_tags_were_fetched(self):
        d = load("pk.json")
        fetched = [p for p in engine.explore(d, "changelog") if p.env.get("no local tags") and p.env.get("a tag does not name a version")]
        self.assertTrue(fetched)                           # a fetch adds tags: what can be wrong with a tag still stands
        self.assertTrue(any(("stamp versions", "a tag does not name a version") in p.open for p in fetched))

    def test_the_code_variant_differs_in_two_places(self):
        got = engine.compare(load("pk.json"), load("pk-code.json"))
        self.assertEqual([f["subject"] for f in got], [
            "branch checks: the design refuses, the code goes ahead",
            "pre-flight: the design refuses, the code goes ahead"])


if __name__ == "__main__":
    unittest.main()
