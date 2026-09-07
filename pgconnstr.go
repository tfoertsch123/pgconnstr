/*
   Copyright 2026 Torsten Foertsch

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 */

// Package pgconnstr implements postgres connection-string parsing and
// rendering. It is not a general purpose syntax checker. It just allows
// you to convert from the URL representation to a key=value style connection
// string and vice versa. It might generate invalid connection strings on
// invalid input.
package pgconnstr

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// PgConnstr is a map of PostgreSQL connection-string keyword/value pairs.
// It is the in-memory representation produced by Parse and consumed by the
// DSN and URL rendering methods. Keys are libpq parameter keywords (such as
// "host", "port", "dbname", "user", and "password"); values are their
// unquoted, unescaped string values.
type PgConnstr map[string]string

// String returns the connection parameters as a libpq keyword/value (DSN)
// string. It is a convenience alias for DSN so that PgConnstr satisfies the
// fmt.Stringer interface.
func (pgc PgConnstr) String() string {
	return pgc.DSN()
}

var order = map[string]int{
	"servicefile": -8,
	"service":     -7,
	"hostaddr":    -6,
	"host":        -5,
	"port":        -4,
	"dbname":      -3,
	"user":        -2,
	"password":    -1,
}

func keywordSort(a, b string) int {
	aorder, aok := order[a]
	border, bok := order[b]
	if aok && bok {
		if aorder < border {
			return -1
		}
		if aorder == border {
			return 0
		}
		return 1
	}
	if aok {
		return -1
	}
	if bok {
		return 1
	}
	return strings.Compare(a, b)
}

var q_re = regexp.MustCompile(`['\\]`)

// DSN renders the connection parameters as a libpq keyword/value (DSN)
// string. Keys are emitted in a canonical order — the well-known libpq
// keywords first, then any remaining keys alphabetically. Values that
// contain a single quote or a backslash are single-quoted and have those
// characters backslash-escaped.
func (pgc PgConnstr) DSN() string {
	keys := slices.SortedFunc(maps.Keys(pgc), keywordSort)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		val := pgc[k]
		qval := q_re.ReplaceAllString(val, `\${0}`)
		if val == qval {
			pairs[i] = k + "=" + val
		} else {
			pairs[i] = k + "='" + qval + "'"
		}
	}
	return strings.Join(pairs, " ")
}

// Parse parses a PostgreSQL connection string into a PgConnstr. It accepts
// either a libpq keyword/value (DSN) string or a connection URL using the
// postgres:// or postgresql:// scheme; the form is detected automatically.
// An error is returned if the input is malformed.
// According to the Postgres documentation, the multi-host URL
// postgres://host1:port1,host2:port2,host3:port3/ is equivalent to the DSN
// "host=host1,host2,host3 port=port1,port2,port3". The spec also allows for
// a single port which applies to all hosts. The corresponding URL would be
// postgres://host1,host2,host3:port. Other than that, the documentation
// requires the same number of elements in the port and host lists and also
// in hostaddr if specified. This module does not validate this condition
// in any format.
func Parse(s string) (PgConnstr, error) {
	if strings.HasPrefix(s, "postgres://") ||
		strings.HasPrefix(s, "postgresql://") {
		return parseURL(s)
	}
	return parseDSN(s)
}

// Errors returned by the DSN parser. Each describes a specific
// malformed-input condition and may be tested for with errors.Is.
var (
	ErrKeyExpected   = errors.New("Key expected")
	ErrDanglingEsc   = errors.New("Dangling escape at end of string")
	ErrNotTerminated = errors.New("Unterminated quoted value")
	ErrExpectingEq   = errors.New("Expecting '=' after key")
)

// parseDSN parses a libpq keyword/value (DSN) connection string into a
// PgConnstr. Whitespace separates pairs, '=' separates a key from its
// value, and values may be single-quoted (with backslash escaping inside
// quotes).
func parseDSN(s string) (PgConnstr, error) {
	i := 0
	n := len(s)

	skipSpace := func() {
		for i < n && unicode.IsSpace(rune(s[i])) {
			i++
		}
	}

	readKey := func() (string, error) {
		start := i
		for i < n {
			c := s[i]
			if c == '=' || unicode.IsSpace(rune(c)) {
				break
			}
			i++
		}
		if start == i {
			return "", fmt.Errorf("%w at pos %d", ErrKeyExpected, start)
		}
		return s[start:i], nil
	}

	readValue := func() (string, error) {
		if i >= n {
			return "", nil
		}

		var b strings.Builder
		if s[i] == '\'' {
			// the key='val ue' case
			i++ // consume opening quote
			for i < n {
				c := s[i]
				if c == '\'' {
					i++ // consume closing quote
					return b.String(), nil
				}
				if c == '\\' {
					i++ // consume backslash
					if i >= n {
						return "",
							fmt.Errorf("%w", ErrDanglingEsc)
					}
				}
				b.WriteByte(s[i])
				i++ // consume character
			}
			return "", fmt.Errorf("%w", ErrNotTerminated)
		}

		// the key=value case
		for i < n {
			c := s[i]
			if unicode.IsSpace(rune(c)) {
				i++ // consume space
				return b.String(), nil
			}
			if c == '\\' {
				i++ // consume backslash
				if i >= n {
					return "",
						fmt.Errorf("%w", ErrDanglingEsc)
				}
			}
			b.WriteByte(s[i])
			i++ // consume character
		}
		return b.String(), nil
	}

	pgc := PgConnstr{}

	for {
		skipSpace()
		if i >= n {
			break
		}

		k, err := readKey()
		if err != nil {
			return nil, err
		}

		skipSpace()

		if i >= n || s[i] != '=' {
			return nil, fmt.Errorf("%w %q at pos %d", ErrExpectingEq, k, i)
		}
		i++ // '='

		skipSpace()

		v, err := readValue()
		if err != nil {
			return nil, err
		}

		pgc[k] = v
	}

	return pgc, nil
}

// Copy returns a shallow copy of the connection parameters. The returned
// PgConnstr is a distinct map: mutating it does not affect the original,
// and mutating the original does not affect the copy.
func (pgc PgConnstr) Copy() PgConnstr {
	other := PgConnstr{}
	for k, v := range pgc {
		other[k] = v
	}
	return other
}

// Redacted returns a copy of the connection parameters with the "password"
// value, if present, replaced by "XXXXXXXX". The original PgConnstr is left
// unmodified. The result is suitable for producing connection strings that
// are safe to log.
func (pgc PgConnstr) Redacted() PgConnstr {
	other := pgc.Copy()
	if _, ok := other["password"]; ok {
		other["password"] = "XXXXXXXX"
	}
	return other
}

// Local Variables:
// tab-width: 4
// End:
