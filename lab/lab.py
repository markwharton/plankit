#!/usr/bin/env python3
"""control lab: a local page for designing a system as a definition file.

It serves one page, runs the control checker on the files it is given,
and runs it again whenever a file changes or the text in the page's
editor changes. It holds no rule of its own: every finding, verdict and
trace comes from `control view` and `control check`.

    python3 lab.py                      the bundled examples and designs/
    python3 lab.py DIR                  every definition file under DIR
    python3 lab.py --root R DEF [DESC ...]   one design in a repository

Standard library only. It listens on 127.0.0.1 and nowhere else.
"""
import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import webbrowser
from html.parser import HTMLParser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

HERE = os.path.dirname(os.path.abspath(__file__))
PAGE = os.path.join(HERE, "lab.html")
sys.path.insert(0, os.path.join(HERE, "engine"))
import engine  # noqa: E402  (reads a design's file, and builds the Go tools the lab runs)
import view as runnable  # noqa: E402  (the engine for designs you can run, and its model for this page)
NAME = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*\.json$")
HEADER = "X-Control-Lab"


class LabError(Exception):
    """A request the lab refuses. The message names what to do."""

    def __init__(self, status, message):
        super().__init__(message)
        self.status = status


# ---------- the checker ----------

class Checker:
    """Runs the control binary. Builds it from its source when it is given
    a source directory, and builds it again when the source is newer: the
    engine's one builder of Go tools does the building."""

    def __init__(self, binary=None, source=None):
        self.binary, self.source = binary, source
        if not binary and not source:
            raise LabError(500, "name the checker: --control <binary>, or --control-src <the control directory>")
        if source and not os.path.isfile(os.path.join(source, "main.go")):
            raise LabError(500, "%s is not the control directory: it has no main.go" % source)

    def ready(self):
        """The path of a binary that is at least as new as its source."""
        if not self.source:
            if not shutil.which(self.binary) and not os.path.isfile(self.binary):
                raise LabError(500, "no control binary at %s" % self.binary)
            return self.binary
        try:
            return engine.go_tool(self.source, "control", self.binary)
        except engine.Refused as e:
            raise LabError(500, "the checker: %s. A checker already built is named with --control <binary>" % e)

    def stamp(self):
        if self.source:
            return engine.source_time(self.source)
        return os.path.getmtime(self.binary) if os.path.isfile(self.binary) else 0.0

    def run(self, args):
        return subprocess.run([self.ready()] + args, capture_output=True, text=True, timeout=60)


class ViewData(HTMLParser):
    """Reads the JSON the view page carries in its script element with
    the id view-data."""

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.inside, self.parts, self.found = False, [], False

    def handle_starttag(self, tag, attrs):
        if tag == "script" and dict(attrs).get("id") == "view-data":
            self.inside = self.found = True

    def handle_endtag(self, tag):
        if tag == "script":
            self.inside = False

    def handle_data(self, data):
        if self.inside:
            self.parts.append(data)


def view_model(html):
    parser = ViewData()
    parser.feed(html)
    if not parser.found:
        raise LabError(500, "the page control view wrote has no view-data element: the lab and the checker no longer agree")
    return json.loads("".join(parser.parts))


# ---------- scenarios ----------

def read_json(path):
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    except (OSError, ValueError):
        return None


def accepted_beside(path):
    return path[:-len(".json")] + ".accepted.json" if path.endswith(".json") else path + ".accepted.json"


class Scenario:
    def __init__(self, definition, descriptions, root, label=None, note="", readonly=False):
        self.definition = os.path.abspath(definition)
        self.descriptions = [os.path.abspath(d) for d in descriptions]
        self.root = os.path.abspath(root)
        self.id = hashlib.sha1(self.definition.encode()).hexdigest()[:12]
        self.label = label or self.shown()
        self.note, self.readonly = note, readonly

    def shown(self):
        """The definition file's path as the page shows it."""
        rel = os.path.relpath(self.definition, self.root)
        return os.path.basename(self.definition) if rel.startswith("..") else rel

    def watched(self):
        """Every file whose change can change this scenario's result: the
        files themselves, their acceptances, and the files their holds
        and sources name."""
        files = [self.definition, accepted_beside(self.definition)]
        for d in self.descriptions:
            files += [d, accepted_beside(d)]
        seen = set()

        def walk(x):
            if isinstance(x, dict):
                f = x.get("file")
                if isinstance(f, str) and ("test" in x or "function" in x):
                    seen.add(os.path.join(self.root, f))
                for v in x.values():
                    walk(v)
            elif isinstance(x, list):
                for v in x:
                    walk(v)

        for path in [self.definition] + self.descriptions:
            walk(read_json(path))
        return files + sorted(seen)

    def stamp(self):
        h = hashlib.sha1()
        for path in self.watched():
            try:
                st = os.stat(path)
                h.update(("%s:%d:%d;" % (path, st.st_mtime_ns, st.st_size)).encode())
            except OSError:
                h.update((path + ":none;").encode())
        return h.hexdigest()[:16]

    def listing(self):
        return {"id": self.id, "label": self.label, "note": self.note, "readonly": self.readonly,
                "file": self.shown(), "root": self.root, "stamp": self.stamp(),
                "descriptions": [os.path.relpath(d, self.root) for d in self.descriptions]}


def scan_directory(directory, readonly=False):
    """Every definition file under a directory. The descriptions of each
    are the description files in its own directory, and that directory
    is its root."""
    found = []
    for base, dirs, names in os.walk(directory):
        dirs[:] = sorted(d for d in dirs if not d.startswith("."))
        docs = {n: read_json(os.path.join(base, n)) for n in sorted(names) if n.endswith(".json") and not n.endswith(".accepted.json")}
        descriptions = [os.path.join(base, n) for n, d in docs.items() if isinstance(d, dict) and "describes" in d]
        for n, d in docs.items():
            if isinstance(d, dict) and "system" in d:
                path = os.path.join(base, n)
                found.append(Scenario(path, descriptions, base, label=os.path.relpath(path, directory), readonly=readonly))
    return found


def read_manifest(path, doc):
    """A manifest lists scenarios whose roots differ from their files'
    directories. Paths in it start from the manifest's own directory."""
    base = os.path.dirname(os.path.abspath(path))
    at = lambda p: os.path.normpath(os.path.join(base, p))
    out = []
    for i, entry in enumerate(doc["scenarios"]):
        for key in ("definition", "root"):
            if key not in entry:
                raise LabError(500, "%s: scenario %d has no %s" % (path, i + 1, key))
        out.append(Scenario(at(entry["definition"]), [at(d) for d in entry.get("descriptions", [])], at(entry["root"]),
                            label=entry.get("label"), note=entry.get("note", ""), readonly=bool(entry.get("readonly"))))
    return out


class Library:
    """The scenarios the lab was started with, read again on each
    listing so that a new file in a scanned directory appears."""

    def __init__(self, sources, root):
        self.sources, self.root = sources, root

    def scenarios(self):
        out, files = [], []
        for src in self.sources:
            if not os.path.exists(src):
                raise LabError(500, "%s is gone: the lab's files were moved or deleted while it was running. Stop the lab and start it again from where it is now" % src)
            if os.path.isdir(src):
                out += scan_directory(src)
                continue
            doc = read_json(src)
            if isinstance(doc, dict) and "scenarios" in doc:
                out += read_manifest(src, doc)
            else:
                files.append((src, doc))
        current = None
        for path, doc in files:
            if isinstance(doc, dict) and "describes" in doc and current is not None:
                current.descriptions.append(os.path.abspath(path))
            else:
                current = Scenario(path, [], self.root or os.getcwd())
                out.append(current)
        seen, unique = set(), []
        for sc in out:
            if sc.id not in seen:
                seen.add(sc.id)
                unique.append(sc)
        return unique

    def get(self, ident):
        for sc in self.scenarios():
            if sc.id == ident:
                return sc
        raise LabError(404, "no scenario with the id %s: the file may have been moved or deleted" % ident)


# ---------- one run of the checker on a scenario ----------

def check_scenario(checker, sc, text=None):
    """Run view and check on a scenario. With text, the definition file
    is taken to hold that text; nothing on disk is changed."""
    try:                                         # a design you can run is played by the engine beside the lab
        if text is None:
            with open(sc.definition, encoding="utf-8") as f:
                text_now = f.read()
        else:
            text_now = text
        doc = json.loads(text_now)
    except (OSError, ValueError) as e:
        doc = None
        if text is not None or not os.path.isfile(sc.definition):
            return {"ok": False, "refused": ["%s: %s" % (sc.shown(), e)]}
    if isinstance(doc, dict) and "parts" in doc:
        return runnable.check(doc, sc.root, sc.shown(), os.path.dirname(sc.definition))   # and where the definitions it stands on are
    work = None
    try:
        definition = sc.definition
        if text is not None:
            work = tempfile.mkdtemp(prefix="control-lab-")
            definition = os.path.join(work, os.path.basename(sc.definition))
            with open(definition, "w", encoding="utf-8") as f:
                f.write(text)
            if os.path.isfile(accepted_beside(sc.definition)):
                shutil.copyfile(accepted_beside(sc.definition), accepted_beside(definition))
        args = ["--root", sc.root, definition] + sc.descriptions
        shown = lambda s: s.replace(definition, sc.shown()).replace(sc.root + os.sep, "")
        view = checker.run(["view"] + args)
        if view.returncode != 0:
            reasons = [shown(line) for line in view.stderr.splitlines() if line.strip()]
            return {"ok": False, "refused": reasons or ["control view exited %d and said nothing" % view.returncode]}
        model = view_model(view.stdout)
        model["definition"] = sc.shown()
        model["descriptions"] = [os.path.relpath(d, sc.root) for d in sc.descriptions]
        check = checker.run(["check"] + args)
        return {"ok": True, "model": model, "viewSummary": shown(view.stderr.strip()),
                "check": {"exit": check.returncode, "lines": [shown(l) for l in check.stdout.splitlines()],
                          "summary": shown(check.stderr.strip())}}
    finally:
        if work:
            shutil.rmtree(work, ignore_errors=True)


def description_prompt(source):
    """The prompt for writing a description file, read from the checker's
    README so that the lab has no copy of it."""
    if not source:
        return {"text": None, "why": "the lab was started without --control-src, so it cannot read the checker's README"}
    path = os.path.join(source, "README.md")
    try:
        with open(path, encoding="utf-8") as f:
            lines = f.read().splitlines()
    except OSError:
        return {"text": None, "why": "%s cannot be read" % path}
    try:
        at = lines.index("## Writing a description file")
        start = next(i for i in range(at, len(lines)) if lines[i].startswith("```"))
        end = next(i for i in range(start + 1, len(lines)) if lines[i].startswith("```"))
    except (ValueError, StopIteration):
        return {"text": None, "why": "%s has no prompt under the heading \"Writing a description file\"" % path}
    return {"text": "\n".join(lines[start + 1:end]), "source": path}


def runnable_prompts():
    """The prompts for a design you can run, read from the engine's own page of them."""
    try:
        with open(os.path.join(HERE, "engine", "PROMPTS.md"), encoding="utf-8") as f:
            lines = f.read().splitlines()
    except OSError:
        return []
    out, title, block = [], None, None
    for line in lines:
        if line.startswith("## "):
            title = line[3:]
        elif line.startswith("```"):
            if block is None:
                block = []
            else:
                out.append({"title": title or "", "text": "\n".join(block)})
                block = None
        elif block is not None:
            block.append(line)
    return out


# ---------- the server ----------

def make_handler(library, checker):
    cache, cache_lock = {}, threading.Lock()

    def result(sc, text=None):
        key = (sc.id, sc.stamp(), checker.stamp(), hashlib.sha1(text.encode()).hexdigest() if text is not None else None)
        with cache_lock:
            if key in cache:
                return cache[key]
        got = check_scenario(checker, sc, text)
        with cache_lock:
            if len(cache) > 64:
                cache.clear()
            cache[key] = got
        return got

    class Handler(BaseHTTPRequestHandler):
        server_version = "control-lab"

        def log_message(self, fmt, *args):
            pass

        def send(self, status, body, kind="application/json"):
            data = body if isinstance(body, bytes) else json.dumps(body).encode()
            self.send_response(status)
            self.send_header("Content-Type", kind + "; charset=utf-8")
            self.send_header("Content-Length", str(len(data)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(data)

        def local(self):
            """Refuse a request that did not come from the lab's own page."""
            host = self.headers.get("Host", "")
            ours = {"127.0.0.1:%d" % self.server.server_port, "localhost:%d" % self.server.server_port}
            if host not in ours:
                raise LabError(403, "the lab answers only on 127.0.0.1")
            origin = self.headers.get("Origin")
            if origin and origin not in {"http://" + h for h in ours}:
                raise LabError(403, "the lab answers only its own page")

        def handle_one(self, fn):
            try:
                self.local()
                fn()
            except LabError as e:
                self.send(e.status, {"error": str(e)})
            except subprocess.TimeoutExpired:
                self.send(500, {"error": "the checker did not finish in 60 seconds"})
            except Exception as e:  # report it to the page; never take the server down
                self.send(500, {"error": "%s: %s" % (type(e).__name__, e)})

        def do_GET(self):
            self.handle_one(self.get)

        def do_POST(self):
            self.handle_one(self.post)

        def get(self):
            url = urlparse(self.path)
            q = {k: v[0] for k, v in parse_qs(url.query).items()}
            if url.path == "/":
                with open(PAGE, "rb") as f:
                    return self.send(200, f.read(), "text/html")
            if url.path == "/api/scenarios":
                return self.send(200, {"scenarios": [s.listing() for s in library.scenarios()], "checker": checker.stamp()})
            if url.path == "/api/model":
                return self.send(200, result(library.get(q.get("id", ""))))
            if url.path == "/api/source":
                sc = library.get(q.get("id", ""))
                with open(sc.definition, encoding="utf-8") as f:
                    return self.send(200, {"text": f.read()})
            if url.path == "/api/prompts":
                return self.send(200, {"description": description_prompt(checker.source), "runnable": runnable_prompts()})
            raise LabError(404, "nothing at %s" % url.path)

        def post(self):
            if self.headers.get(HEADER) != "1":
                raise LabError(403, "a change must come from the lab's page, with the header %s" % HEADER)
            length = int(self.headers.get("Content-Length") or 0)
            if length > 4 * 1024 * 1024:
                raise LabError(413, "the text is over 4 MB")
            try:
                body = json.loads(self.rfile.read(length) or b"{}")
            except ValueError:
                raise LabError(400, "the request is not JSON")
            if not isinstance(body, dict) or not isinstance(body.get("text"), str):
                raise LabError(400, "the request needs the definition's text")
            sc = library.get(body.get("id", ""))
            if self.path == "/api/check":
                return self.send(200, result(sc, body["text"]))
            if self.path == "/api/save":
                return self.send(200, self.save(sc, body))
            raise LabError(404, "nothing at %s" % self.path)

        def save(self, sc, body):
            if sc.readonly:
                raise LabError(403, "%s is marked read-only: take a snapshot, or open your own copy of the file" % sc.label)
            name = body.get("name")
            if name is None:
                target = sc.definition
            else:
                if not isinstance(name, str) or not NAME.match(name) or name.endswith(".accepted.json"):
                    raise LabError(400, "a new file's name is letters, digits, dots, dashes and underscores, ends .json, and has no directory")
                target = os.path.join(os.path.dirname(sc.definition), name)
                if os.path.exists(target):
                    raise LabError(409, "%s exists: choose another name, or open it and save there" % name)
            with open(target, "w", encoding="utf-8") as f:
                f.write(body["text"])
            saved = [s for s in library.scenarios() if s.definition == os.path.abspath(target)]
            return {"saved": os.path.relpath(target, sc.root), "id": saved[0].id if saved else None}

    return Handler


class Server(ThreadingHTTPServer):
    daemon_threads = True

    def handle_error(self, request, client_address):
        """A browser opens spare connections and drops them. That is its
        business and is not reported; any other error still is."""
        if isinstance(sys.exc_info()[1], (ConnectionResetError, BrokenPipeError, ConnectionAbortedError)):
            return
        super().handle_error(request, client_address)


def serve(sources, root=None, control=None, control_src=None, port=0):
    checker = Checker(control, control_src)
    checker.ready()
    library = Library(sources, root)
    server = Server(("127.0.0.1", port), make_handler(library, checker))
    return server, library


def main(argv=None):
    ap = argparse.ArgumentParser(description="control lab: design a system as a definition file, with the checker run on every change")
    ap.add_argument("paths", nargs="*", help="a directory of designs, a manifest, or a definition file followed by its description files")
    ap.add_argument("--root", help="the repository's root, which the paths in holds and sources start from (for files named directly)")
    ap.add_argument("--control", help="the control binary to run")
    ap.add_argument("--control-src", help="the control directory to build the checker from (default: ../control beside the lab)")
    ap.add_argument("--port", type=int, default=0, help="the port to listen on (default: any free port)")
    ap.add_argument("--no-open", action="store_true", help="do not open the browser")
    args = ap.parse_args(argv)
    src = args.control_src or (None if args.control else os.path.normpath(os.path.join(HERE, "..", "control")))
    paths = args.paths or [os.path.join(HERE, "examples", "lab.json"), os.path.join(HERE, "designs")]
    for p in paths:
        if not os.path.exists(p):
            print("control lab: %s does not exist" % p, file=sys.stderr)
            return 1
    try:
        server, library = serve(paths, args.root, args.control, src, args.port)
        count = len(library.scenarios())
    except LabError as e:
        print("control lab: %s" % e, file=sys.stderr)
        return 2
    if count == 0:
        print("control lab: no definition file found in %s" % ", ".join(paths), file=sys.stderr)
        return 2
    url = "http://127.0.0.1:%d/" % server.server_port
    print("control lab: %d designs at %s  (Ctrl-C stops it)" % (count, url), file=sys.stderr)
    if not args.no_open:
        webbrowser.open(url)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    return 0


if __name__ == "__main__":
    sys.exit(main())
