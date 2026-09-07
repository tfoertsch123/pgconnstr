package pgconnstr

import (
	"errors"
	"testing"
)

// URL parse tests.

func TestParseURLBasic(t *testing.T) {
	pgc, err := Parse("postgres://localhost:5432/mydb#ignoredFragment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}

func TestParseURLEmptyHost(t *testing.T) {
	pgc, err := Parse("postgres:///mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := pgc["host"]; ok {
		t.Errorf("unexpected host = %q", got)
	}
	if got, ok := pgc["port"]; ok {
		t.Errorf("unexpected port = %q", got)
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}

func TestParseURLOnlyPort(t *testing.T) {
	pgc, err := Parse("postgres://:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := pgc["host"]; ok {
		t.Errorf("unexpected host = %q", got)
	}
	if got, ok := pgc["port"]; !ok || got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}
func TestParseURLOnlyPassword(t *testing.T) {
	pgc, err := Parse("postgres://:5432@/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := pgc["host"]; ok {
		t.Errorf("unexpected host = %q", got)
	}
	if got, ok := pgc["port"]; ok {
		t.Errorf("unexpected port = %q", got)
	}
	if got, ok := pgc["user"]; ok {
		t.Errorf("unexpected user = %q", got)
	}
	if got, ok := pgc["password"]; !ok || got != "5432" {
		t.Errorf("password = %q, want %q", got, "5432")
	}
}

func TestParseURLPostgresqlScheme(t *testing.T) {
	pgc, err := Parse("postgresql://localhost/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
	if got, ok := pgc["port"]; ok {
		t.Errorf("port = %q, should not exist", got)
	}
}

func TestParseURLUserPassword(t *testing.T) {
	pgc, err := Parse("postgres://alice:sec%2Fret@localhost/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "alice" {
		t.Errorf("user = %q, want %q", got, "alice")
	}
	if got := pgc["password"]; got != "sec/ret" {
		t.Errorf("password = %q, want %q", got, "sec/ret")
	}
}

func TestParseURLUserNoPassword(t *testing.T) {
	pgc, err := Parse("postgres://alice@localhost/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "alice" {
		t.Errorf("user = %q, want %q", got, "alice")
	}
	if _, ok := pgc["password"]; ok {
		t.Errorf("password should not be set, got %q", pgc["password"])
	}
}

func TestParseURLNoPort(t *testing.T) {
	pgc, err := Parse("postgres://localhost/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLQueryParams(t *testing.T) {
	pgc, err := Parse("postgres://localhost/mydb?sslmode=disable&application_name=my%20app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["sslmode"]; got != "disable" {
		t.Errorf("sslmode = %q, want %q", got, "disable")
	}
	if got := pgc["application_name"]; got != "my app" {
		t.Errorf("application_name = %q, want %q", got, "my app")
	}
}

func TestParseURLQueryParamOverrides(t *testing.T) {
	// Last value wins for repeated query params.
	pgc, err := Parse("postgres://localhost/mydb?port=5432&port=5433")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["port"]; got != "5433" {
		t.Errorf("port = %q, want %q", got, "5433")
	}
}

func TestParseURLQueryParamOverrides2(t *testing.T) {
	// Last value wins for repeated query params.
	pgc, err := Parse("postgres://localhost:5434/mydb?port=5432&port=5433")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["port"]; got != "5434" {
		t.Errorf("port = %q, want %q", got, "5434")
	}
}

func TestParseURLQueryOverridesPath(t *testing.T) {
	pgc, err := Parse("postgres://localhost/mydb?dbname=other")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The path takes precedence over the query param.
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}

func TestParseURLNoPath(t *testing.T) {
	pgc, err := Parse("postgres://localhost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "localhost" {
		t.Errorf("host = %q, want %q", got, "localhost")
	}
	if _, ok := pgc["dbname"]; ok {
		t.Errorf("dbname should not be set, got %q", pgc["dbname"])
	}
}

func TestParseURLRootPath(t *testing.T) {
	// A bare "/" path should not set dbname.
	pgc, err := Parse("postgres://localhost/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := pgc["dbname"]; ok {
		t.Errorf("dbname should not be set for path /, got %q", pgc["dbname"])
	}
}

func TestParseURLEmptyQueryValue(t *testing.T) {
	pgc, err := Parse("postgres://localhost/mydb?sslmode=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := pgc["sslmode"]; !ok || got != "" {
		t.Errorf("sslmode = %q, want empty", got)
	}
}

func TestParseURLMultipleHostsSinglePort(t *testing.T) {
	pgc, err := Parse("postgres://host1,host2:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2" {
		t.Errorf("host = %q, want %q", got, "host1,host2")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseURLMultipleHostsMultiplePorts(t *testing.T) {
	pgc, err := Parse("postgres://host1:5432,host2:5433/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2" {
		t.Errorf("host = %q, want %q", got, "host1,host2")
	}
	if got := pgc["port"]; got != "5432,5433" {
		t.Errorf("port = %q, want %q", got, "5432,5433")
	}
}

func TestParseURLEmptyHP(t *testing.T) {
	pgc, err := Parse("postgres://host1:5432,,host2:5433/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,,host2" {
		t.Errorf("host = %q, want %q", got, "host1,,host2")
	}
	if got := pgc["port"]; got != "5432,,5433" {
		t.Errorf("port = %q, want %q", got, "5432,,5433")
	}
}

func TestParseURLMultipleHostsNoPort(t *testing.T) {
	pgc, err := Parse("postgres://host1,host2/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2" {
		t.Errorf("host = %q, want %q", got, "host1,host2")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLQueryKeyWithoutValue(t *testing.T) {
	pgc, err := Parse("postgres:///mydb?a=b&c&d=e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := pgc["c"]; !ok || got != "" {
		t.Errorf("c = %q, want %q", got, "")
	}
	if got, ok := pgc["a"]; !ok || got != "b" {
		t.Errorf("a = %q, want %q", got, "b")
	}
	if got, ok := pgc["d"]; !ok || got != "e" {
		t.Errorf("d = %q, want %q", got, "e")
	}
}

func TestParseURLInvalidQuery(t *testing.T) {
	_, err := parseURL("postgres://localhost:5432/mydb?a=b;c=d")
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseURLInvalidDB(t *testing.T) {
	_, err := parseURL("postgres://localhost:5432/my%AXdb")
	if !errors.Is(err, ErrInvalidDB) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseURLInvalidUser(t *testing.T) {
	_, err := parseURL("postgres://ab%AX@localhost:5432/mydb")
	if !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseURLInvalidPW(t *testing.T) {
	_, err := parseURL("postgres://user:pass%AXword@localhost:5432/mydb")
	if !errors.Is(err, ErrInvalidPW) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseURLNonPg(t *testing.T) {
	_, err := parseURL("http://localhost:5432/mydb")
	if !errors.Is(err, ErrNotPg) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseURLInvalid(t *testing.T) {
	_, err := Parse("postgres://[::1")
	if !errors.Is(err, ErrUntermintatedIPv6) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLTooManyColons(t *testing.T) {
	_, err := Parse("postgres://host:54:32")
	if !errors.Is(err, ErrTooManyColons) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLTooManyColonsIPv6(t *testing.T) {
	_, err := Parse("postgres://[::1]:54:32")
	if !errors.Is(err, ErrTooManyColons) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLInvalidIPv6Port(t *testing.T) {
	_, err := Parse("postgres://[::1]5432")
	if !errors.Is(err, ErrMissingColon) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLInvalidHost(t *testing.T) {
	_, err := Parse("postgres://local%AXhost")
	if !errors.Is(err, ErrInvalidHost) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLInvalidHostWithPort(t *testing.T) {
	_, err := Parse("postgres://local%AXhost:123")
	if !errors.Is(err, ErrInvalidHost) {
		t.Fatalf("expected error for invalid URL, got %v", err)
	}
}

func TestParseURLSingleIPv6(t *testing.T) {
	pgc, err := Parse("postgres://[::1]/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1]" {
		t.Errorf("host = %q, want %q", got, "[::1]")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}

func TestParseURLSingleIPv6WithPort(t *testing.T) {
	pgc, err := Parse("postgres://[::1]:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1]" {
		t.Errorf("host = %q, want %q", got, "[::1]")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseURLMultipleIPv6Hosts(t *testing.T) {
	pgc, err := Parse("postgres://[::1],[2001:db8::1]/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],[2001:db8::1]" {
		t.Errorf("host = %q, want %q", got, "[::1],[2001:db8::1]")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLMultipleIPv6HostsWithPort(t *testing.T) {
	pgc, err := Parse("postgres://[::1]:5432,[2001:db8::1]:5433/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],[2001:db8::1]" {
		t.Errorf("host = %q, want %q", got, "[::1],[2001:db8::1]")
	}
	if got := pgc["port"]; got != "5432,5433" {
		t.Errorf("port = %q, want %q", got, "5432,5433")
	}
}

func TestParseURLEmptyHostWithPort(t *testing.T) {
	pgc, err := Parse("postgres://:1234/dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if got := pgc["port"]; got != "1234" {
		t.Errorf("port = %q, want %q", got, "1234")
	}
	if got := pgc["dbname"]; got != "dbname" {
		t.Errorf("dbname = %q, want %q", got, "dbname")
	}
}

func TestParseURLUserAtEmptyHostWithPort(t *testing.T) {
	pgc, err := Parse("postgres://user@:1234/dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "user" {
		t.Errorf("user = %q, want %q", got, "user")
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if got := pgc["port"]; got != "1234" {
		t.Errorf("port = %q, want %q", got, "1234")
	}
	if got := pgc["dbname"]; got != "dbname" {
		t.Errorf("dbname = %q, want %q", got, "dbname")
	}
}

func TestParseURLUserPassAtEmptyHostWithPort(t *testing.T) {
	pgc, err := Parse("postgres://user:password@:1234/dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "user" {
		t.Errorf("user = %q, want %q", got, "user")
	}
	if got := pgc["password"]; got != "password" {
		t.Errorf("password = %q, want %q", got, "password")
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if got := pgc["port"]; got != "1234" {
		t.Errorf("port = %q, want %q", got, "1234")
	}
}

func TestParseURLNoHostNoPort(t *testing.T) {
	pgc, err := Parse("postgres:///dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
	if got := pgc["dbname"]; got != "dbname" {
		t.Errorf("dbname = %q, want %q", got, "dbname")
	}
}

func TestParseURLUserNoHostNoPort(t *testing.T) {
	pgc, err := Parse("postgres://user@/dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "user" {
		t.Errorf("user = %q, want %q", got, "user")
	}
	if _, ok := pgc["password"]; ok {
		t.Errorf("password should not be set, got %q", pgc["password"])
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
	if got := pgc["dbname"]; got != "dbname" {
		t.Errorf("dbname = %q, want %q", got, "dbname")
	}
}

func TestParseURLUserPassNoHostNoPort(t *testing.T) {
	pgc, err := Parse("postgres://user:password@/dbname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["user"]; got != "user" {
		t.Errorf("user = %q, want %q", got, "user")
	}
	if got := pgc["password"]; got != "password" {
		t.Errorf("password = %q, want %q", got, "password")
	}
	if _, ok := pgc["host"]; ok {
		t.Errorf("host should not be set, got %q", pgc["host"])
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
	if got := pgc["dbname"]; got != "dbname" {
		t.Errorf("dbname = %q, want %q", got, "dbname")
	}
}

func TestParseURLEncodedHost(t *testing.T) {
	pgc, err := Parse("postgres://%2Frun%2Fpostgres:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "/run/postgres" {
		t.Errorf("host = %q, want %q", got, "/run/postgres")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
	if got := pgc["dbname"]; got != "mydb" {
		t.Errorf("dbname = %q, want %q", got, "mydb")
	}
}

func TestParseURLEncodedHostNoPort(t *testing.T) {
	pgc, err := Parse("postgres://%2Frun%2Fpostgres/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "/run/postgres" {
		t.Errorf("host = %q, want %q", got, "/run/postgres")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLMultipleHostsMixed(t *testing.T) {
	pgc, err := Parse("postgres://[::1]:5432,%2Frun%2Fpostgres,host3/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],/run/postgres,host3" {
		t.Errorf("host = %q, want %q", got, "[::1],/run/postgres,host3")
	}
	if got := pgc["port"]; got != "5432,," {
		t.Errorf("port = %q, want %q", got, "5432,,")
	}
}

func TestParseURLMultipleHostsAllIPv6WithPorts(t *testing.T) {
	pgc, err := Parse("postgres://[::1]:5432,[2001:db8::1]:5433/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],[2001:db8::1]" {
		t.Errorf("host = %q, want %q", got, "[::1],[2001:db8::1]")
	}
	if got := pgc["port"]; got != "5432,5433" {
		t.Errorf("port = %q, want %q", got, "5432,5433")
	}
}

func TestParseURLMultipleHostsSingleIPv6Port(t *testing.T) {
	pgc, err := Parse("postgres://[::1],[2001:db8::1]:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],[2001:db8::1]" {
		t.Errorf("host = %q, want %q", got, "[::1],[2001:db8::1]")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseURLMultipleHostsNoPortsAllIPv6(t *testing.T) {
	pgc, err := Parse("postgres://[::1],[2001:db8::1]/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "[::1],[2001:db8::1]" {
		t.Errorf("host = %q, want %q", got, "[::1],[2001:db8::1]")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLMultipleEncodedHosts(t *testing.T) {
	pgc, err := Parse("postgres://%2Frun%2Fpostgres,%2Ftmp%2Fpostgres/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "/run/postgres,/tmp/postgres" {
		t.Errorf("host = %q, want %q", got, "/run/postgres,/tmp/postgres")
	}
	if _, ok := pgc["port"]; ok {
		t.Errorf("port should not be set, got %q", pgc["port"])
	}
}

func TestParseURLUnterminatedIPv6(t *testing.T) {
	if _, err := Parse("postgres://[::1/mydb"); err == nil {
		t.Fatal("expected error for unterminated IPv6 brackets, got nil")
	}
}

func TestParseURLMultipleHostsGarbageAfterBracket(t *testing.T) {
	if _, err := Parse("postgres://[::1]abc/mydb"); err == nil {
		t.Fatal("expected error for garbage after ']', got nil")
	}
}

func TestParseURLMultipleHostsSinglePortOnLast(t *testing.T) {
	// A single port on the last host should produce a bare port value,
	// matching libpq behaviour.
	pgc, err := Parse("postgres://host1,host2:5432/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2" {
		t.Errorf("host = %q, want %q", got, "host1,host2")
	}
	if got := pgc["port"]; got != "5432" {
		t.Errorf("port = %q, want %q", got, "5432")
	}
}

func TestParseURLMultipleHostsSinglePortOnFirst(t *testing.T) {
	// A single port on a non-last host produces a comma-aligned list.
	pgc, err := Parse("postgres://host1:5432,host2/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2" {
		t.Errorf("host = %q, want %q", got, "host1,host2")
	}
	if got := pgc["port"]; got != "5432," {
		t.Errorf("port = %q, want %q", got, "5432,")
	}
}

func TestParseURLMultipleHostsSinglePortOnMiddle(t *testing.T) {
	pgc, err := Parse("postgres://host1:5432,host2,host3/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pgc["host"]; got != "host1,host2,host3" {
		t.Errorf("host = %q, want %q", got, "host1,host2,host3")
	}
	if got := pgc["port"]; got != "5432,," {
		t.Errorf("port = %q, want %q", got, "5432,,")
	}
}

// URL() method tests

func TestURLEmpty(t *testing.T) {
	pgc := PgConnstr{}
	want := "postgres://"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLHostOnly(t *testing.T) {
	pgc := PgConnstr{"host": "localhost"}
	want := "postgres://localhost"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLHostOnlyIPv6(t *testing.T) {
	pgc := PgConnstr{"host": "[::1]"}
	want := "postgres://[::1]"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLHostAndPort(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "port": "5432"}
	want := "postgres://localhost:5432"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLDbnameOnly(t *testing.T) {
	pgc := PgConnstr{"dbname": "mydb"}
	want := "postgres:///mydb"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLUserOnly(t *testing.T) {
	pgc := PgConnstr{"user": "alice"}
	want := "postgres://alice@"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLUserAndPassword(t *testing.T) {
	pgc := PgConnstr{"user": "alice", "password": "secret"}
	want := "postgres://alice:secret@"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLPasswordNoUser(t *testing.T) {
	// A password with no user still emits an empty user segment.
	pgc := PgConnstr{"password": "secret"}
	want := "postgres://:secret@"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLPortNoHost(t *testing.T) {
	// A port with no host is dropped; there is nothing to attach it to.
	pgc := PgConnstr{"port": "5432"}
	want := "postgres://:5432"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLMultiPortNoHost(t *testing.T) {
	// A port with no host is dropped; there is nothing to attach it to.
	pgc := PgConnstr{"port": "5432,5433,5434"}
	want := "postgres://:5432,:5433,:5434"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLFull(t *testing.T) {
	pgc := PgConnstr{
		"host":     "localhost",
		"port":     "5432",
		"dbname":   "mydb",
		"user":     "me",
		"password": "secret",
	}
	want := "postgres://me:secret@localhost:5432/mydb"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLQueryParams(t *testing.T) {
	pgc := PgConnstr{"host": "localhost", "sslmode": "disable"}
	want := "postgres://localhost?sslmode=disable"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLKnownKeysExcludedFromQuery(t *testing.T) {
	// host/port/dbname/user/password must never appear in the query string.
	pgc := PgConnstr{
		"host":     "localhost",
		"port":     "5432",
		"dbname":   "mydb",
		"user":     "me",
		"password": "secret",
		"sslmode":  "disable",
	}
	want := "postgres://me:secret@localhost:5432/mydb?sslmode=disable"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLQueryParamOrdering(t *testing.T) {
	// Extra params are sorted by keywordSort: known keys first (none here),
	// then the rest alphabetically.
	pgc := PgConnstr{"host": "localhost", "zebra": "z", "apple": "a"}
	want := "postgres://localhost?apple=a&zebra=z"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLPercentEscaping(t *testing.T) {
	// Spaces and slashes in user/dbname are percent-encoded.
	pgc := PgConnstr{"user": "a b/c", "host": "localhost", "dbname": "my db"}
	want := "postgres://a%20b%2Fc@localhost/my%20db"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLIPv6Host(t *testing.T) {
	pgc := PgConnstr{"host": "[::1]", "port": "5432", "dbname": "mydb"}
	want := "postgres://[::1]:5432/mydb"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLMultipleHostsMultiplePorts(t *testing.T) {
	pgc := PgConnstr{
		"host":   "host1,[::1],/run/pg",
		"port":   "5432,5433,5432",
		"dbname": "mydb",
	}
	want := "postgres://host1:5432,[::1]:5433,%2Frun%2Fpg:5432/mydb"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLMultipleHostsSinglePort(t *testing.T) {
	pgc := PgConnstr{"host": "host1,[::1],/var/run/postgresql", "port": "5432"}
	want := "postgres://host1,[::1],%2Fvar%2Frun%2Fpostgresql:5432"
	if got := pgc.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestURLRoundTrip(t *testing.T) {
	orig := "postgres://me:secret@localhost:5432/mydb?sslmode=disable"
	pgc, err := Parse(orig)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	out := pgc.URL()
	if out != orig {
		t.Errorf("URL() round trip =\n  %q\nwant\n  %q", out, orig)
	}
}

func TestURLRoundTripDbnameOnly(t *testing.T) {
	orig := "postgres:///mydb"
	pgc, err := Parse(orig)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	out := pgc.URL()
	if out != orig {
		t.Errorf("URL() round trip =\n  %q\nwant\n  %q", out, orig)
	}
}

func TestURLRoundTripMultipleHostsPorts(t *testing.T) {
	orig := "postgres://host1:5432,host2:5433/mydb"
	pgc, err := Parse(orig)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	out := pgc.URL()
	if out != orig {
		t.Errorf("URL() round trip =\n  %q\nwant\n  %q", out, orig)
	}
}

func TestURLRoundTripService(t *testing.T) {
	orig := "postgres:///bench?servicefile=..%2Fpgservice.conf&service=mydb"
	pgc, err := Parse(orig)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	out := pgc.URL()
	if out != orig {
		t.Errorf("URL() round trip =\n  %q\nwant\n  %q", out, orig)
	}
}

// Local Variables:
// tab-width: 4
// End:
