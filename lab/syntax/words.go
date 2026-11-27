package main

import "strings"

// Form is how a design is written as text. Both forms have the same lines and the same words;
// they differ in how a keyword is told from a name.
type Form string

const (
	// A writes keywords in capitals and names as they are: changelog WHEN NOT a release is pending.
	A Form = "A"
	// B writes keywords in lower case and every name in square brackets: [changelog] when not [a release is pending].
	B Form = "B"
)

// keywords is the one list of the words of the text form, in lower case. The lexers read a word
// as one of these, and the printer writes from it. A keyword of two words is two of these in a row.
var keywords = []string{
	"system", "on", "reports", "faults", "runs",
	"place", "one", "many", "series", "outside", "lasting", "run", "report", "in", "given", "owner",
	"fact", "fault", "open", "about", "only", "means", "at",
	"says", "because", "hold",
	"seq", "all", "choose", "when", "otherwise", "nothing", "action", "as",
	"uses", "with", "reads", "create", "replace", "destroy", "from", "makes", "later", "undone", "stands", "rehearsal",
	"refuse", "fail", "warn", "tell", "ask", "accept", "no", "stop", "skip", "tells",
	"expect", "never", "changes", "creates", "ends", "before", "is", "success", "failed", "declined",
	"untouched", "created", "replaced", "destroyed", "changed", "unchanged",
	"not", "and", "or", "any",
	"rule",
}

// texts are the keywords whose line runs on as text to its end.
var texts = map[string]bool{"system": true, "says": true, "because": true, "rule": true}

var isKeyword = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range keywords {
		m[w] = true
	}
	return m
}()

// kw is a keyword as the form writes it.
func (f Form) kw(w string) string {
	if f == A {
		return strings.ToUpper(w)
	}
	return w
}

// keyword is the keyword a word of this form is, if it is one.
func (f Form) keyword(word string) (string, bool) {
	w := strings.ToLower(word)
	if !isKeyword[w] || f.kw(w) != word {
		return "", false
	}
	return w, true
}
