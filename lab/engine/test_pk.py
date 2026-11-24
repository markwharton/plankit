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
        self.assertEqual(collections.Counter(p.run for p in plays), {"changelog": 79, "release": 52, "ship": 233})
        found = engine.findings(d, plays)
        self.assertEqual(collections.Counter(f["kind"] for f in found), {"expect": 1, "unanswered": 5, "left-behind": 2})
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
