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
        return engine.Design(json.load(f), REPO)


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
        # 364 plays before places had a shape. Each fact a place brings is played both ways where a change brings it.
        self.assertEqual(collections.Counter(p.run for p in plays), {"changelog": 101, "release": 62, "ship": 301})
        found = engine.findings(d, plays)
        # Converted by convert.py, the design asks eight things more: what happens to what was there in each of
        # the seven lasting places it replaces, and whether a fetched tag is one the repository already has.
        self.assertEqual(collections.Counter(f["kind"] for f in found), {"expect": 1, "unanswered": 13, "left-behind": 2})
        brought = sorted(f["subject"] for f in found if f["subject"].endswith(("holds what is kept nowhere", "already has the member")))
        self.assertEqual(brought, ["changelog file holds what is kept nowhere", "origin holds what is kept nowhere", "release branch holds what is kept nowhere",
                                   "release commit holds what is kept nowhere", "remote-tracking branches holds what is kept nowhere", "tags already has the member",
                                   "version files holds what is kept nowhere", "working tree holds what is kept nowhere"])
        self.assertIn(("expect", "changelog with --dry-run writes nothing"), [(f["kind"], f["subject"]) for f in found])
        self.assertIn(("unanswered", "settings are invalid"), [(f["kind"], f["subject"]) for f in found])
        self.assertNotIn(("unanswered", "the tree is dirty"), [(f["kind"], f["subject"]) for f in found])   # --undo refuses a dirty tree

    def test_the_code_variant_differs_in_two_places(self):
        got = engine.compare(load("pk.json"), load("pk-code.json"))
        self.assertEqual([f["subject"] for f in got], [
            "branch checks: the design refuses, the code goes ahead",
            "pre-flight: the design refuses, the code goes ahead"])


if __name__ == "__main__":
    unittest.main()
