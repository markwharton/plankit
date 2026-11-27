// The two lexers of a design written as text. make syntax generates lexer.go from this file with
// ragel -Z (Go), takes out the //line directives ragel still writes, so the file does not change
// with ragel's version, and formats it with gofmt.
//
// A lexer reads one line, with its indentation already taken off. A word is read as a keyword
// when the form writes that keyword so (words.go); a keyword whose line runs on as text stops
// the machine, and the rest of the line is the text.

package main

%%{
	machine lexA;
	word = [^ \t()=\[\]]+;
	main := |*
		word => { if l.word(ts, te) { fbreak; } };
		'(' => { l.add(tOpen, ts, te) };
		')' => { l.add(tClose, ts, te) };
		'=' => { l.add(tEq, ts, te) };
		' '+;
	*|;
	write data;
}%%

%%{
	machine lexB;
	name = '[' [^\]\t]* ']';
	word = [^ \t()=,\[\]]+;
	main := |*
		name => { l.add(tName, ts+1, te-1) };
		word => { if l.word(ts, te) { fbreak; } };
		'(' => { l.add(tOpen, ts, te) };
		')' => { l.add(tClose, ts, te) };
		'=' => { l.add(tEq, ts, te) };
		',' => { l.add(tComma, ts, te) };
		' '+;
	*|;
	write data;
}%%

func lexLineA(l *lexer) (int, bool) {
	data := l.data
	cs, p, pe, eof := 0, 0, len(data), len(data)
	ts, te, act := 0, 0, 0
	_, _, _ = ts, te, act
	%% machine lexA;
	%% write init;
	%% write exec;
	return p, cs != lexA_error
}

func lexLineB(l *lexer) (int, bool) {
	data := l.data
	cs, p, pe, eof := 0, 0, len(data), len(data)
	ts, te, act := 0, 0, 0
	_, _, _ = ts, te, act
	%% machine lexB;
	%% write init;
	%% write exec;
	return p, cs != lexB_error
}

