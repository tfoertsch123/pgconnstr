package pgconnstr

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
)

// Errors returned by the URL parser. Each describes a specific
// malformed-input condition and may be tested for with errors.Is.
var (
	ErrNotPg             = errors.New("Not a postgres URL")
	ErrInvalidQuery      = errors.New("Invalid query")
	ErrInvalidDB         = errors.New("Invalid DB name")
	ErrInvalidUser       = errors.New("Invalid user name")
	ErrInvalidPW         = errors.New("Invalid password")
	ErrInvalidHost       = errors.New("Invalid host")
	ErrTooManyColons     = errors.New("Too many colons")
	ErrMissingColon      = errors.New("Missing colon")
	ErrUntermintatedIPv6 = errors.New("Unterminated IPv6 address")
)

// parseURL parses a postgres connection URL (postgres:// or postgresql://)
// into a PgConnstr. Unlike net/url.Parse, it supports:
//   - multiple comma-separated hosts, each optionally with its own port,
//     including bracketed IPv6 literals ([::1],[2001:db8::1]:5432,...);
//   - an empty host with a port (postgres://:5432/db);
//   - a percent-encoded host (e.g. a Unix socket path such as
//     postgres://%2Frun%2Fpostgres:5432/db);
//   - no host at all (postgres:///db, postgres://user@/db).
func parseURL(s string) (PgConnstr, error) {
	var rest string
	switch {
	case strings.HasPrefix(s, "postgres://"):
		rest = s[len("postgres://"):]
	case strings.HasPrefix(s, "postgresql://"):
		rest = s[len("postgresql://"):]
	default:
		return nil, fmt.Errorf("%w: %q", ErrNotPg, s)
	}

	// Drop fragment.
	if i := strings.Index(rest, "#"); i >= 0 {
		rest = rest[:i]
	}

	// Split off query string.
	var query string
	if i := strings.Index(rest, "?"); i >= 0 {
		query = rest[i+1:]
		rest = rest[:i]
	}

	// Split authority (everything up to the first '/') and path.
	var authority, path string
	if i := strings.Index(rest, "/"); i >= 0 {
		authority = rest[:i]
		path = rest[i:]
	} else {
		authority = rest
	}

	pgc := PgConnstr{}

	// Query parameters: last value wins for repeated keys.
	if query != "" {
		vals, err := url.ParseQuery(query)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
		}
		for k, vs := range vals {
			pgc[k] = vs[len(vs)-1]
		}
	}

	// Database name from path (percent-decoded).
	if len(path) > 1 { // skip the leading '/'
		db, err := url.PathUnescape(path[1:])
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidDB, path[1:])
		}
		pgc["dbname"] = db
	}

	// Userinfo: everything up to the last '@'.
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		userinfo := authority[:at]
		authority = authority[at+1:]
		user, pass, ok := strings.Cut(userinfo, ":")
		if len(user) > 0 {
			u, err := url.PathUnescape(user)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidUser, user)
			}
			pgc["user"] = u
		}
		if ok {
			p, err := url.PathUnescape(pass)
			if err != nil {
				return nil, fmt.Errorf("%w", ErrInvalidPW)
			}
			pgc["password"] = p
		}
	}

	// Hosts: comma-separated, each host[:port]. An IPv6 literal is bracketed.
	if authority != "" {
		parts := strings.Split(authority, ",")
		hosts := make([]string, len(parts))
		ports := make([]string, len(parts))
		nports := 0
		for i, hp := range parts {
			h, p, err := splitHostPort(hp)
			if err != nil {
				return nil, err
			}
			hosts[i] = h
			ports[i] = p
			if p != "" {
				nports++
			}
		}
		joined := strings.Join(hosts, ",")
		if joined != "" {
			pgc["host"] = joined
		}
		switch nports {
		case 0:
			// No ports given.
		default:
			// One or more ports. If exactly one port is given and it belongs
			// to the last host, libpq uses a bare port value. Otherwise the
			// ports are aligned with hosts (empty = default) as a comma list.
			if nports == 1 && ports[len(ports)-1] != "" {
				pgc["port"] = ports[len(ports)-1]
			} else {
				pgc["port"] = strings.Join(ports, ",")
			}
		}
	}

	return pgc, nil
}

// splitHostPort splits a single "host:port" authority segment. IPv6 literals
// must be bracketed ([::1]); the brackets are preserved in the returned host.
// The host part of a non-bracketed segment is percent-decoded so that
// percent-encoded values (e.g. Unix socket paths) are supported.
func splitHostPort(hp string) (host, port string, err error) {
	if hp == "" {
		return "", "", nil
	}
	if hp[0] == '[' {
		end := strings.Index(hp, "]")
		if end < 0 {
			return "", "", fmt.Errorf("%w in %q", ErrUntermintatedIPv6, hp)
		}
		host = hp[:end+1] // keep the brackets: [::1]
		tail := hp[end+1:]
		if tail == "" {
			return host, "", nil
		}
		if tail[0] != ':' {
			return "", "", fmt.Errorf("%w in %q", ErrMissingColon, hp)
		}
		if strings.Contains(tail[1:], ":") {
			return "", "", fmt.Errorf("%w in %q", ErrTooManyColons, hp)
		}
		return host, tail[1:], nil
	}
	// Non-bracketed: the first ':' separates host and port. An encoded colon
	// (%3A) is not a separator, so splitting on a literal ':' is safe.
	if i := strings.Index(hp, ":"); i >= 0 {
		h, p := hp[:i], hp[i+1:]
		if strings.Contains(p, ":") {
			return "", "", fmt.Errorf("%w in %q", ErrTooManyColons, hp)
		}
		host, e := url.PathUnescape(h)
		if e != nil {
			return "", "", fmt.Errorf("%w: %v", ErrInvalidHost, h)
		}
		return host, p, nil
	}
	host, e := url.PathUnescape(hp)
	if e != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidHost, hp)
	}
	return host, "", nil
}

// URL renders the connection parameters as a PostgreSQL connection URL
// using the postgres:// scheme. The userinfo (user and password), host and
// port, and database name are emitted in their conventional URL positions
// and percent-encoded as needed; IPv6 host literals keep their brackets.
// Any remaining parameters are appended as a query string, ordered by the
// same canonical keyword sort as DSN.
// According to the Postgres documentation, the multi-host URL
// postgres://host1:port1,host2:port2,host3:port3/ is equivalent to the DSN
// "host=host1,host2,host3 port=port1,port2,port3". The spec also allows for
// a single port which applies to all hosts. The corresponding URL would be
// postgres://host1,host2,host3:port. Other than that, the documentation
// requires the same number of elements in the port and host lists and also
// in hostaddr if specified. This module does not check that and might generate
// invalid URLs on invalid input.
func (pgc PgConnstr) URL() string {
	userinfo := ""
	if x, ok := pgc["user"]; ok {
		userinfo = url.PathEscape(x)
	}
	if x, ok := pgc["password"]; ok {
		userinfo += ":" + url.PathEscape(x)
	}
	if len(userinfo) > 0 {
		userinfo += "@"
	}

	hostport := ""
	if x, ok := pgc["host"]; ok {
		if y, ok := pgc["port"]; ok {
			hparts := strings.Split(x, ",")
			pparts := strings.Split(y, ",")
			if len(pparts) == 1 {
				parts := make([]string, len(hparts))
				for i, hp := range hparts {
					if hp[0] == '[' && hp[len(hp)-1] == ']' {
						// IPv6 address
						parts[i] = hp
					} else {
						parts[i] = url.PathEscape(hp)
					}
				}
				hostport = strings.Join(parts, ",") + ":" + url.PathEscape(y)
			} else {
				nparts := max(len(pparts), len(hparts))
				parts := make([]string, nparts)
				for i := range nparts {
					hp := ""
					if i < len(hparts) {
						hp = hparts[i]
						if hp[0] != '[' || hp[len(hp)-1] != ']' {
							// not an IPv6 address
							hp = url.PathEscape(hp)
						}
					}
					if i < len(pparts) {
						hp += ":" + url.PathEscape(pparts[i])
					}
					parts[i] = hp
				}
				hostport = strings.Join(parts, ",")
			}
		} else {
			if x[0] == '[' && x[len(x)-1] == ']' {
				hostport = x // assuming IPv6
			} else {
				hostport = url.PathEscape(x)
			}
		}
	} else if y, ok := pgc["port"]; ok {
		// only port given
		pparts := strings.Split(y, ",")
		parts := make([]string, len(pparts))
		for i, p := range pparts {
			parts[i] = ":" + url.PathEscape(p)
		}
		hostport = strings.Join(parts, ",")
	}

	u := "postgres://" + userinfo + hostport
	if x, ok := pgc["dbname"]; ok {
		u += "/" + url.PathEscape(x)
	}

	keys := slices.SortedFunc(maps.Keys(pgc), keywordSort)
	pairs := make([]string, len(keys))
	j := 0
FOR:
	for _, k := range keys {
		switch k {
		case "user", "password", "host", "port", "dbname":
			continue FOR
		}
		// url.QueryEscape encodes a space as +. Postgres does not like that.
		// It wants %20.
		pairs[j] = strings.ReplaceAll(url.QueryEscape(k), "+", "%20") +
			"=" + strings.ReplaceAll(url.QueryEscape(pgc[k]), "+", "%20")
		j++
	}
	if j > 0 {
		u += "?" + strings.Join(pairs[:j], "&")
	}

	return u
}

// Local Variables:
// tab-width: 4
// End:
