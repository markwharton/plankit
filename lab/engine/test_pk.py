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
    return engine.Design(engine.read_design(os.path.join(HERE, name)), REPO, HERE)


class Pk(unittest.TestCase):
    def test_sixteen_of_the_21_statements_are_played_and_five_stay_words(self):
        with open(os.path.join(REPO, "definition", "pk.json"), encoding="utf-8") as f:
            words = [u["what"] for u in json.load(f)["unsaid"]]
        stated = {}
        for s, how in engine.said(load("pk.design")):
            if s in words:
                stated[s] = "played" if how == "played" or stated.get(s) == "played" else "words"
        self.assertEqual(len(words), 21)
        self.assertEqual(collections.Counter(stated.values()), {"played": 16, "words": 5})
        self.assertEqual([w for w in words if w not in stated], [])

    def test_the_plays_and_the_questions_are_the_ones_the_page_gives(self):
        d = load("pk.design")
        plays = engine.explore_all(d)
        self.assertEqual(collections.Counter(p.run for p in plays), {"changelog": 150, "release": 78, "ship": 349})
        found = [(f["kind"], f["subject"]) for f in engine.findings(d, plays)]
        self.assertEqual(found, [])

    def test_what_git_goes_on_with_pk_refuses(self):
        d = load("pk.design")
        plays = engine.explore(d, "changelog")
        for fault in ("working branch is held elsewhere", "working tree holds what is kept nowhere"):   # git's reset accepts both
            mine = [p for p in plays if p.env.get("undo") and p.env.get(fault)]
            self.assertTrue(mine, fault)
            self.assertEqual({(p.outcome(), tuple(p.steps)) for p in mine}, {("failed", (("load settings", "ran"), ("undo", "refused")))}, fault)

    def test_the_code_variant_differs_in_23_places(self):
        got = engine.compare(load("pk.design"), load("pk-code.design"))
        self.assertEqual([f["subject"] for f in got], [
            "branch checks: the design refuses, the code goes ahead",                                # a branch behind origin: changelog makes no check
            "branch checks: the design skips it, the code goes ahead",                               # in ship, a trailer that is not a version: the design counts it pending, the code runs changelog
            "branch checks: the design tells and goes on, the code goes ahead",                      # no default named: changelog says nothing
            "build section: the design goes ahead, the code has no such action",                     # no commit is in a visible section: nothing to release
            "commits since tag ends untouched in the design and created in the code",                # the code measures the commits before it reads --bump
            "fetch the tags: the design tells and goes on, the code does its work and then fails",   # a rejected tag update
            "fetch the working branch: the design goes ahead, the code has no such action",          # release checks the branch and the pending release before it fetches
            "fetch the working branch: the design goes ahead, the code skips it",                    # release's dry run reads origin's branch; the design still fetches
            "fetch the working branch: the design refuses, the code has no such action",
            "infer version: the design goes ahead, the code has no such action",                     # nothing to release is said before --bump and --exclude are read
            "infer version: the design refuses, the code goes ahead",                                # every commit excluded: nothing to release, and --bump is not read
            "infer version: the design refuses, the code has no such action",                        # the same with nothing to release; and a tag outside history, refused at measure
            "infer version: the design tells and goes on, the code has no such action",              # an unmatched --exclude is not reached when there is nothing to release
            "load settings: the design goes ahead, the code has no such action",                     # release reads the current branch before the settings
            "load settings: the design goes ahead, the code refuses",                                # not a git repository, or no settings file
            "pending release check: the design goes ahead, the code has no such action",             # every commit excluded: nothing to release
            "pending release check: the design refuses, the code has no such action",                # every commit excluded: said before the pending check
            "pre-flight: the design goes ahead, the code refuses",                                   # merge flow: origin's release branch diverged, or no release branch anywhere
            "pre-flight: the design refuses, the code goes ahead",                                   # origin has the tag on another commit; a protected branch; a release branch that cannot be fast-forwarded, found at the merge
            "pre-flight: the design refuses, the code tells and goes on",                            # the same, where origin also names no default branch
            "read origin's tags: the design goes ahead, the code refuses",                           # a dry run's last tag outside history: the code refuses as it reads origin's tags, the design a step later
            "undo: the design goes ahead, the code refuses",                                         # undo refuses a trailer that is not a version
            "write section: the design tells and goes on, the code goes ahead"])                     # no web address: changelog says nothing


if __name__ == "__main__":
    unittest.main()
