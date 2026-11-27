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
        self.assertEqual(collections.Counter(p.run for p in plays), {"changelog": 148, "release": 69, "ship": 312})
        found = [(f["kind"], f["subject"]) for f in engine.findings(d, plays)]
        self.assertEqual(found, [
            ("unanswered", "origin has no address"),                           # changelog accepts it, and a run that reads what changelog left is asked again
            ("unanswered", "origin's tags already has the member")])           # comes up from git's push, and no push action answers it

    def test_what_git_goes_on_with_pk_refuses(self):
        d = load("pk.json")
        plays = engine.explore(d, "changelog")
        for fault in ("working branch is held elsewhere", "working tree holds what is kept nowhere"):   # git's reset accepts both
            mine = [p for p in plays if p.env.get("undo") and p.env.get(fault)]
            self.assertTrue(mine, fault)
            self.assertEqual({(p.outcome(), tuple(p.steps)) for p in mine}, {("failed", (("load settings", "ran"), ("undo", "refused")))}, fault)

    def test_the_code_variant_differs_in_five_places(self):
        got = engine.compare(load("pk.json"), load("pk-code.json"))
        self.assertEqual([f["subject"] for f in got], [
            "branch checks: the design refuses, the code goes ahead",                       # a branch behind origin: changelog makes no check
            "branch checks: the design tells and goes on, the code goes ahead",             # no default named: changelog says nothing
            "fetch the tags: the design skips it, the code does its work and then fails",    # a dry run: the design reads origin's tags and fetches nothing
            "fetch the tags: the design skips it, the code goes ahead",
            "fetch the tags: the design tells and goes on, the code does its work and then fails"])   # a rejected tag update


if __name__ == "__main__":
    unittest.main()
