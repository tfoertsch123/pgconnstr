# TEST_VERBOSE:=1
# TEST_VERBOSE:=

V=$(if $(findstring 1,$(TEST_VERBOSE)),-v)

test:
	@go test $V -count=1 -coverprofile cover.out ./
	@go tool cover -html=cover.out -o coverage.html
	@echo Coverage report in file://$$PWD/coverage.html

doc:
	pkgsite -http=127.0.0.1:6060

# update VERSION before running make tag
tag:	VERSION
	git fetch origin && \
	[ "$$(git branch --show-current)" = master ] && \
	L="$$(git rev-parse HEAD)" && \
	R="$$(git rev-parse refs/remotes/origin/master)" && \
	[ "$$L" = "$$R" ] && \
	T="$$(cat VERSION)" && \
	git tag $$T && \
	git push origin tag $$T && \
	M="$$(sed '/^module/!d; s/^module *//' go.mod)" && \
	GOPROXY=proxy.golang.org go list -m $$M@$$T
