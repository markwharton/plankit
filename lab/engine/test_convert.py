"""Tests of the conversion of a design written before places had a shape.

    python3 -m unittest test_convert     (from this directory)
"""
import unittest

import convert


def old():
    return {
        "system": "contacts", "reports": "message",
        "places": [{"name": "file", "kind": "outside"}, {"name": "book", "kind": "lasting", "keyed": True}, {"name": "index", "kind": "lasting"},
                   {"name": "count", "kind": "run"}, {"name": "message", "kind": "report"}],
        "facts": [{"name": "book already has the record", "about": "book", "says": "a second import is normal"}],
        "parts": {"import": {"seq": [
            {"do": "read", "reads": ["file"], "changes": ["count"]},
            {"do": "add", "reads": ["count"], "changes": [{"place": "book", "how": "create", "many": True}, "index"],
             "handles": [{"when": {"any": ["book already has the record", "book holds another's record"]}, "then": "refuse"}]},
            {"choose": [{"when": "book already has the record", "run": {"do": "drop", "changes": [{"place": "index", "how": "delete"}, {"place": "book"}]}}], "otherwise": None},
        ]}},
        "runs": ["import"],
        "expects": [{"name": "an import ends with the index written", "ends": {"index": "set", "count": "set"}},
                    {"name": "the book is there before the index", "before": "add", "place": "index", "is": "deleted"}],
    }


class Convert(unittest.TestCase):
    def test_a_keyed_place_holds_many_and_any_other_holds_one(self):
        self.assertEqual([(p["name"], p["is"], "keyed" in p) for p in convert.convert(old())["places"]],
                         [("file", "one", False), ("book", "many", False), ("index", "one", False), ("count", "one", False), ("message", "one", False)])

    def test_each_change_says_how(self):
        seq = convert.convert(old())["parts"]["import"]["seq"]
        self.assertEqual(seq[0]["changes"], [{"place": "count", "how": "create"}])                       # a place of one run gets its first state
        self.assertEqual(seq[1]["changes"], [{"place": "book", "how": "create", "many": True}, {"place": "index", "how": "replace"}])
        self.assertEqual(seq[2]["choose"][0]["run"]["changes"], [{"place": "index", "how": "destroy"}, {"place": "book", "how": "create"}])   # set on a keyed place adds

    def test_what_a_keyed_place_brought_is_named_for_members(self):
        new = convert.convert(old())
        self.assertEqual(new["facts"][0]["name"], "book already has the member")
        self.assertEqual(new["parts"]["import"]["seq"][1]["handles"][0]["when"], {"any": ["book already has the member", "book holds another's member"]})
        self.assertEqual(new["parts"]["import"]["seq"][2]["choose"][0]["when"], "book already has the member")

    def test_a_state_in_an_expectation_follows_the_same_rule(self):
        e = convert.convert(old())["expects"]
        self.assertEqual((e[0]["ends"], e[1]["is"]), ({"index": "replaced", "count": "created"}, "destroyed"))

    def test_a_design_with_shapes_is_left_as_it_is(self):
        doc = old()
        once = convert.convert(doc)
        self.assertIsNot(once, doc)
        self.assertIs(convert.convert(once), once)
        self.assertEqual(doc, old())                       # and the design given is not changed


if __name__ == "__main__":
    unittest.main()
