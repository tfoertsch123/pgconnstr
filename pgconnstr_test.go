package pgconnstr

import (
	"errors"
	"testing"
)

func TestParseEmpty(t *testing.T) {
	pgc, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pgc) != 0 {
		t.Fatalf("expected empty map, got %v", pgc)
	}
}

func TestParseWhitespaceOnly(t *testing.T) {
	pgc, err := Parse("   \t\n  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pgc) != 0 {
		t.Fatalf("expected empty map, got %v", pgc)
	}
}

func TestParseSimple(t *testing.T) {
	pgc, err := Parse("host=localhost port=5432")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseExtraWhitespace(t *testing.T) {
	pgc, err := Parse("  host  =  localhost   port  =  '5432'  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseQuotedValue(t *testing.T) {
	pgc, err := Parse("password='secret'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["password"]; got != "secret" {
		t.Errorf("password = %q, want %q", got, "secret")
	}
}

func TestParseQuotedValueWithSpaces(t *testing.T) {
	pgc, err := Parse("dbname='my database'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["dbname"]; got != "my database" {
		t.Errorf("dbname = %q, want %q", got, "my database")
	}
}

func TestParseEscapedQuote(t *testing.T) {
	// The escaped quote inside a value is preserved literally.
	pgc, err := Parse(`password='pa\'ss\\ word'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["password"]; got != `pa'ss\ word` {
		t.Errorf("password = <%s>, want <%s>", got, `pa'ss\ word`)
	}
}

func TestParseTrailingEscape(t *testing.T) {
	if _, err := Parse("password='abc\\"); !errors.Is(err, ErrDanglingEsc) {
		t.Fatalf("expected error for dangling escape, got %v", err)
	}
}

func TestParseTrailingEscape2(t *testing.T) {
	if _, err := Parse("password=abc\\"); !errors.Is(err, ErrDanglingEsc) {
		t.Fatalf("expected error for dangling escape, got %v", err)
	}
}

func TestParseUnterminatedQuote(t *testing.T) {
	if _, err := Parse("password='abc"); !errors.Is(err, ErrNotTerminated) {
		t.Fatalf("expected error for unterminated quote, got %v", err)
	}
}

func TestParseMissingEquals(t *testing.T) {
	if _, err := Parse("host"); !errors.Is(err, ErrExpectingEq) {
		t.Fatalf("expected error for missing '=', got %v", err)
	}
}

func TestParseMissingEqualsWithValue(t *testing.T) {
	if _, err := Parse("host localhost"); err == nil {
		t.Fatalf("expected error for missing '=', got %v", err)
	}
}

func TestParseEmptyKey(t *testing.T) {
	if _, err := Parse("=value"); !errors.Is(err, ErrKeyExpected) {
		t.Fatalf("expected error for empty key, got %v", err)
	}
}

func TestParseEmptyValue(t *testing.T) {
	pgc, err := Parse("hugo = erwin host=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "" {
		t.Errorf("host = %q, want empty", got)
	}
}

func TestParseEmptyValue2(t *testing.T) {
	_, err := Parse(" host= hugo = erwin")
	if err == nil {
		t.Fatal("unexpected success")
	}
}

func TestParseEmptyQuotedValue(t *testing.T) {
	pgc, err := Parse("host='' hugo = erwin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "" {
		t.Errorf("host = %q, want %q", got, "")
	}
}

func TestParseEscapedBackslash(t *testing.T) {
	// Backslash escaping just quotes the next char. It does not have to be
	// a space or a backslash
	pgc, err := Parse(`host = 'a\b'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != `ab` {
		t.Errorf("host = <%s>, want <%s>", got, `ab`)
	}
}

func TestParseBackslashUnquoted(t *testing.T) {
	// Backslash escaping just quotes the next char. It does not have to be
	// a space or a backslash
	pgc, err := Parse(`host = a\b hugo=erwin`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != `ab` {
		t.Errorf("host = <%s>, want <%s>", got, `ab`)
	}
}

func TestDSNOrdering(t *testing.T) {
	pgc := PgConnstr{
		"dbname":   "test",
		"user":     "me",
		"host":     "localhost",
		"port":     "5432",
		"password": "secret",
	}
	dsn := pgc.DSN()
	// Known keys should appear in the canonical order: host, port, dbname, user, password
	want := "host=localhost port=5432 dbname=test user=me password=secret"
	if dsn != want {
		t.Errorf("DSN =\n  %q\nwant\n  %q", dsn, want)
	}
}

func TestDSNUnknownKeysAlphabetical(t *testing.T) {
	pgc := PgConnstr{
		"zebra": "z",
		"apple": "a",
		"host":  "localhost",
	}
	dsn := pgc.DSN()
	want := "host=localhost apple=a zebra=z"
	if dsn != want {
		t.Errorf("DSN =\n  %q\nwant\n  %q", dsn, want)
	}
}

func TestDSNKnownBeforeUnknown(t *testing.T) {
	pgc := PgConnstr{
		"extra": "x",
		"port":  "5432",
	}
	dsn := pgc.DSN()
	want := "port=5432 extra=x"
	if dsn != want {
		t.Errorf("DSN =\n  %q\nwant\n  %q", dsn, want)
	}
}

func TestDSNQuotingApostrophe(t *testing.T) {
	pgc := PgConnstr{
		"password": "it's a back\\slash",
	}
	dsn := pgc.DSN()
	want := `password='it\'s a back\\slash'`
	if dsn != want {
		t.Errorf("DSN = %q, want %q", dsn, want)
	}
}

func TestDSNNoQuotingWhenNoApostrophe(t *testing.T) {
	pgc := PgConnstr{
		"host": "localhost",
	}
	dsn := pgc.DSN()
	want := "host=localhost"
	if dsn != want {
		t.Errorf("DSN = %q, want %q", dsn, want)
	}
}

func TestStringEqualsDSN(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "port": "5432"}
	if pgc.String() != pgc.DSN() {
		t.Error("String() should equal DSN()")
	}
}

func TestKeywordSortKnownOnly(t *testing.T) {
	keys := []string{"port", "host", "user", "hostaddr", "password",
		"dbname", "service", "servicefile", "dbname"}
	// sort using keywordSort
	sorted := append([]string(nil), keys...)
	// manual insertion sort using keywordSort
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && keywordSort(sorted[j], sorted[j-1]) < 0; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	want := []string{"servicefile", "service", "hostaddr", "host", "port",
		"dbname", "dbname", "user", "password"}
	for i, k := range want {
		if sorted[i] != k {
			t.Errorf("sorted[%d] = %q, want %q (full: %v)", i, sorted[i], k, sorted)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	orig := "host=localhost port=5432 dbname=test user=me password=secret"
	pgc, err := Parse(orig)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	round := pgc.DSN()
	if round != orig {
		t.Errorf("round trip =\n  %q\nwant\n  %q", round, orig)
	}
}

func TestCopyEmpty(t *testing.T) {
	pgc := PgConnstr{}
	cp := pgc.Copy()
	if len(cp) != 0 {
		t.Fatalf("expected empty copy, got %v", cp)
	}
}

func TestCopyIsShallow(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "port": "5432"}
	cp := pgc.Copy()
	if len(cp) != len(pgc) {
		t.Fatalf("copy len = %d, want %d", len(cp), len(pgc))
	}
	for k, v := range pgc {
		if cp[k] != v {
			t.Errorf("copy[%q] = %q, want %q", k, cp[k], v)
		}
	}
}

func TestCopyIsIndependent(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "port": "5432"}
	cp := pgc.Copy()
	cp["host"] = "remote"
	cp["extra"] = "x"
	if pgc["host"] != "localhost" {
		t.Errorf("original host mutated: %q", pgc["host"])
	}
	if _, ok := pgc["extra"]; ok {
		t.Errorf("original gained key %q", "extra")
	}
}

func TestCopyDoesNotShareMap(t *testing.T) {
	// Mutating the original after Copy must not affect the copy.
	pgc := PgConnstr{"host": "localhost"}
	cp := pgc.Copy()
	pgc["port"] = "5432"
	if _, ok := cp["port"]; ok {
		t.Errorf("copy gained key %q from later original mutation", "port")
	}
}

func TestRedactedNoPassword(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "user": "me"}
	r := pgc.Redacted()
	if _, ok := r["password"]; ok {
		t.Errorf("password should not be set, got %q", r["password"])
	}
	for k, v := range pgc {
		if r[k] != v {
			t.Errorf("redacted[%q] = %q, want %q", k, r[k], v)
		}
	}
}

func TestRedactedReplacesPassword(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "user": "me", "password": "secret"}
	r := pgc.Redacted()
	if got := r["password"]; got != "XXXXXXXX" {
		t.Errorf("redacted password = %q, want %q", got, "XXXXXXXX")
	}
	if r["host"] != "localhost" {
		t.Errorf("host = %q, want %q", r["host"], "localhost")
	}
	if r["user"] != "me" {
		t.Errorf("user = %q, want %q", r["user"], "me")
	}
}

func TestRedactedDoesNotMutateOriginal(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "password": "secret"}
	_ = pgc.Redacted()
	if pgc["password"] != "secret" {
		t.Errorf("original password mutated: %q", pgc["password"])
	}
}

func TestRedactedEmpty(t *testing.T) {
	pgc := PgConnstr{}
	r := pgc.Redacted()
	if len(r) != 0 {
		t.Fatalf("expected empty redacted, got %v", r)
	}
}

func TestRedactedDSNDoesNotLeakPassword(t *testing.T) {
	// A common use case: produce a safe DSN string for logging.
	pgc := PgConnstr{"host": "localhost", "user": "me", "password": "secret"}
	want := "host=localhost user=me password=XXXXXXXX"
	if got := pgc.Redacted().DSN(); got != want {
		t.Errorf("redacted DSN =\n  %q\nwant\n  %q", got, want)
	}
}

// Local Variables:
// tab-width: 4
// End:
