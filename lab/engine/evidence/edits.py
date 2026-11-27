# ---- Q1
@edit
def q1_true_pk_accepts(pk, git):
    git["facts"] = [f for f in git["facts"] if f["name"] != "a rejected tag update fails the fetch"]
    git["parts"]["fetch tags"]["handles"] = [
        {"when": "origin does not answer", "then": "refuse"},
        {"when": "tags already has the member", "then": "fail", "says": "The update is rejected, the other tags are created, and git exits 1."},
        {"when": "tags is given a key twice", "then": "accept", "because": "A remote holds one tag of each name."}]
@edit
def q1_true_pk_refuses(pk, git):
    q1_true_pk_accepts(pk, git)
    h = part(pk, "fetch the tags")["handles"]
    h[1] = {"when": "tags already has the member", "then": "refuse", "says": "no version tags found locally: git fetch failed; origin has tags; fetch them: git fetch --tags"}
@edit
def q1_true_pk_silent(pk, git):
    q1_true_pk_accepts(pk, git)
    h = part(pk, "fetch the tags")["handles"]
    del h[1]
# ---- Q2
@edit
def q2_narrow_expectation(pk, git):
    e = find(pk["expects"], "name", "changelog with --dry-run writes nothing")
    e["never changes"] = ["working tree", "origin", "release branch", "working branch"]
    e["because"] = "a dry run fetches origin's tags, as a real run does, so that the section it prints is the one a real run writes"
@edit
def q2_dry_run_reads_origin(pk, git):
    seq = part(pk, "make changelog")["seq"]
    seq[1] = {"use": "fetch the tags", "when": {"all": ["no local tags", {"not": "dry run"}]}}
    seq.insert(2, {"use": "read origin's tags", "when": {"all": ["no local tags", "dry run"]}})
    pk["parts"]["read origin's tags"] = {"do": "read origin's tags", "reads": ["origin's tags"],
        "handles": [{"when": "origin does not answer", "then": "refuse"}], "says": "git ls-remote --tags origin"}
    iv = part(pk, "infer version")
    iv["reads"] = ["tags", {"place": "origin's tags", "when": {"all": ["no local tags", "dry run"]}}, "working branch"]
# ---- Q3
@edit
def q3_accept_as_code(pk, git):
    part(pk, "infer version")["handles"].insert(0, {"when": "a tag does not name a version", "then": "accept",
        "because": "a tag that is not a version is not a release: version inference passes over it",
        "hold": {"file": "internal/changelog/changelog_test.go", "test": "TestANonVersionTagIsNotARelease"}})
    pf = part(pk, "pre-flight")["handles"][0]["when"]["any"]
    pf.remove("a tag does not name a version")
    part(pk, "pre-flight")["handles"].insert(0, {"when": "a tag does not name a version", "then": "accept",
        "because": "release measures the last tag the way changelog does, and passes over a tag that is not a version"})
@edit
def q3_refuse(pk, git):
    part(pk, "infer version")["handles"].insert(0, {"when": "a tag does not name a version", "then": "refuse"})
# ---- Q4
@edit
def q4_state_not_fault(pk, git):
    f = fact(pk, "origin has no address"); f.pop("fault")
    f["says"] = "origin's URL gives no web address, so the section is written with no link"
@edit
def q4_refuse(pk, git):
    h = part(pk, "write section")["handles"]
    h[1] = {"when": "origin has no address", "then": "refuse"}
# ---- Q5
@edit
def q5_push_refuses(pk, git):
    for p in ("push in trunk flow", "push in merge flow"):
        part(pk, p)["handles"][0]["when"]["any"].append("origin's tags already has the member")
@edit
def q5_preflight_refuses(pk, git):
    pf = part(pk, "pre-flight")
    pf["reads"].append("origin's tags")
    pf["handles"].append({"when": "origin's tags already has the member", "then": "refuse"})
@edit
def q5_same_commit_is_fine(pk, git):
    git["facts"].append({"name": "origin's tag names the commit pushed", "says": "origin already has the tag, on the commit the push sends", "only": "origin's tags already has the member"})
    git["parts"]["push the branch and the tag"]["handles"].insert(2, {"when": "origin's tag names the commit pushed", "then": "accept", "because": "the ref is already what the push sends, so git takes the push"})
    q5_push_refuses(pk, git)
# ---- Q6
@edit
def q6_load_settings_first(pk, git):
    pk["parts"]["load settings"] = {"do": "load settings", "reads": ["settings"], "handles": [{"when": "settings are invalid", "then": "refuse",
        "says": "An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: pk commands exit 2"}]}
    pk["parts"]["changelog"] = {"name": "changelog run", "seq": ["load settings", {"choose": [{"when": "undo", "run": "undo"}], "otherwise": "make changelog"}]}
    part(pk, "release")["seq"].insert(0, "load settings")
    for p in ("branch checks", "pre-flight"):
        part(pk, p)["handles"][0]["when"]["any"].remove("settings are invalid")
# ---- Q7 / Q8: split the checks the way the code makes them
@edit
def q78_split_as_code(pk, git):
    q6_load_settings_first(pk, git)
    pk["facts"] = [f for f in pk["facts"] if f["name"] != "the working branch is unfit"]
    pk["facts"] += [
        {"name": "no branch is checked out", "fault": True, "about": "working branch"},
        {"name": "the working branch is protected", "fault": True, "about": "working branch"},
        {"name": "the working branch is not on origin", "fault": True, "about": "working branch"},
        {"name": "the working branch is behind origin", "fault": True, "about": "working branch", "only": {"not": "in changelog"}}]
    bc = part(pk, "branch checks"); bc.pop("when")
    bc["handles"] = [
        {"when": {"all": [{"not": "dry run"}, {"any": ["working tree holds what is kept nowhere", "no branch is checked out"]}]}, "then": "refuse"},
        {"when": {"any": ["the working branch is protected", "the branch is not the default", "the working branch is not on origin", "origin does not answer"]}, "then": "refuse"},
        {"when": {"any": ["working tree holds what is kept nowhere", "no branch is checked out"]}, "then": "accept", "because": "a dry run writes nothing, so it may read the commits from anywhere"}]
    pf = part(pk, "pre-flight")["handles"][0]["when"]["any"]
    pf.remove("the working branch is unfit"); pf += ["no branch is checked out", "the working branch is protected", "the working branch is not on origin", "the working branch is behind origin"]
    part(pk, "undo")["handles"].append({"when": {"any": ["the branch is not the default", "no branch is checked out", "the working branch is protected", "the working branch is not on origin"]},
        "then": "accept", "because": "undo removes only a commit that nothing else holds, so the branch it is on does not matter"})
# ---- Q9 / Q10
@edit
def q9_stands(pk, git):
    p = part(pk, "fetch the working branch"); p["later"] = "stands"
    p["because"] = "a remote-tracking branch is a copy of origin's branch, and the next fetch makes it again"
@edit
def q10_stands(pk, git):
    p = part(pk, "fetch the tags"); p["later"] = "stands"
    p["because"] = "the tags fetched are origin's, and a later run would fetch them again"
@edit
def q10_undone(pk, git):
    p = part(pk, "fetch the tags"); p["later"] = "undone"
@edit
def q3_state_with_rule(pk, git):
    f = fact(pk, "a tag does not name a version"); f.pop("fault")
    f["says"] = "the repository has a tag whose name is not a version"
    part(pk, "pre-flight")["handles"][0]["when"]["any"].remove("a tag does not name a version")
    pk["rules"].append({"rule": "version inference passes over a tag that is not a version", "about": "infer version", "when": "a tag does not name a version",
        "hold": {"file": "internal/changelog/changelog_test.go", "test": "TestANonVersionTagIsNotARelease"}})
@edit
def q78_behind_refused(pk, git):
    q78_split_as_code(pk, git)
    fact(pk, "the working branch is behind origin").pop("only")
    part(pk, "branch checks")["handles"][1]["when"]["any"].append("the working branch is behind origin")
@edit
def q78_behind_accepted(pk, git):
    q78_split_as_code(pk, git)
    fact(pk, "the working branch is behind origin").pop("only")
    part(pk, "branch checks")["handles"].insert(0, {"when": "the working branch is behind origin", "then": "accept", "because": "release refuses a branch behind origin before it tags"})
def _undo_accepts_behind(pk):
    part(pk, "undo")["handles"][-1]["when"]["any"].append("the working branch is behind origin")
@edit
def q78_as_code(pk, git):
    q78_behind_accepted(pk, git); _undo_accepts_behind(pk)
@edit
def q78_refuse_everywhere(pk, git):
    q78_behind_refused(pk, git); _undo_accepts_behind(pk)
@edit
def all_as_code(pk, git):
    q1_true_pk_refuses(pk, git); q2_narrow_expectation(pk, git); q3_state_with_rule(pk, git); q4_state_not_fault(pk, git)
    q5_push_refuses(pk, git); q78_as_code(pk, git); q9_stands(pk, git); q10_stands(pk, git)
@edit
def x_preflight_lets_dirty_tree_through(pk, git):
    part(pk, "pre-flight")["handles"][0]["when"]["any"].remove("working tree holds what is kept nowhere")
@edit
def x_merge_decided(pk, git):
    git["facts"] = [f for f in git["facts"] if f["name"] != "a merge refuses a tree with changes it would overwrite"]
    git["parts"]["fast-forward"]["handles"][1] = {"when": "working tree holds what is kept nowhere", "then": "refuse",
        "says": "git merge --ff-only refuses, exit 1, when a changed or untracked file is in the way; it goes on, keeping them, when none is"}
@edit
def q1_true_pk_fails(pk, git):
    q1_true_pk_accepts(pk, git)
    part(pk, "fetch the tags")["handles"][1] = {"when": "tags already has the member", "then": "fail"}
