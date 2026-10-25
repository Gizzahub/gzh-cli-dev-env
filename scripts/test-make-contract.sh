#!/bin/bash
# Exercise the actual repository recipes without installing or running tools.
set -euo pipefail

root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$root/tmp"
fixture=$(mktemp -d "$root/tmp/make-contract.XXXXXX")
cleanup() {
    local status=$?
    if [ "$status" -ne 0 ]; then
        cat "$fixture/stdout" "$fixture/stderr" >&2
    fi
    rm -f -- "$fixture/override.mk" "$fixture/go" "$fixture/formatter" \
        "$fixture/trace" "$fixture/stdout" "$fixture/stderr" "$fixture/sample.go"
    rmdir -- "$fixture"
}
trap cleanup EXIT
export CONTRACT_TRACE="$fixture/trace"
export CONTRACT_FAIL_STAGE=""

cat > "$fixture/go" <<'EOF'
#!/bin/sh
if [ "$1" = env ]; then
    exit 0
fi
printf 'go:%s\n' "$*" >> "$CONTRACT_TRACE"
EOF
cat > "$fixture/formatter" <<'EOF'
#!/bin/sh
printf 'formatter:%s\n' "$*" >> "$CONTRACT_TRACE"
EOF
chmod +x "$fixture/go" "$fixture/formatter"
export PATH="$fixture:$PATH"
printf 'package fixture\n' > "$fixture/sample.go"

# The explicit MAKE variable propagates overrides to recursive invocations.
cat > "$fixture/override.mk" <<'EOF'
.PHONY: format-check lint-check test make-contract-test clean build install-format-tools install-gofumpt install-goimports install-gci install-golangci-lint
format-check lint-check test make-contract-test:
	@printf '%s\n' '$@' >> "$$CONTRACT_TRACE"
	@if [ "$$CONTRACT_FAIL_STAGE" = '$@' ]; then echo 'contract-stage-failure:$@' >&2; exit 23; fi
clean build install-format-tools install-gofumpt install-goimports install-gci install-golangci-lint:
	@:
EOF

invoke() {
    : > "$CONTRACT_TRACE"
    if make --no-print-directory -C "$root" -f Makefile -f "$fixture/override.mk" \
        MAKE="make --no-print-directory -f Makefile -f $fixture/override.mk" \
        "$@" > "$fixture/stdout" 2> "$fixture/stderr"; then
        result=0
    else
        result=$?
    fi
}
expect_trace() {
    local expected=$1
    local actual
    actual=$(cat "$CONTRACT_TRACE")
    if [ "$actual" != "$expected" ]; then
        printf 'Unexpected trace\nexpected: %s\nactual: %s\n' "$expected" "$actual" >&2
        exit 1
    fi
}

invoke definitely-absent-contract-goal
[ "$result" -ne 0 ]
grep -q 'definitely-absent-contract-goal' "$fixture/stderr"
expect_trace ""

invoke -j4 check
[ "$result" -eq 0 ]
expect_trace $'format-check\nlint-check\ntest\nmake-contract-test'

expected=""
for stage in format-check lint-check test make-contract-test; do
    expected="${expected}${expected:+$'\n'}$stage"
    export CONTRACT_FAIL_STAGE="$stage"
    invoke -j4 check
    [ "$result" -ne 0 ]
    grep -q "contract-stage-failure:$stage" "$fixture/stderr"
    expect_trace "$expected"
done
export CONTRACT_FAIL_STAGE=""

invoke VERSION=contract run positional-argument
[ "$result" -eq 0 ]
expect_trace 'go:run -ldflags -X main.version=contract ./cmd/gzh-git positional-argument'

invoke VERSION=contract ARGS=variable-argument run
[ "$result" -eq 0 ]
expect_trace 'go:run -ldflags -X main.version=contract ./cmd/gzh-git variable-argument'

invoke GOFUMPT="$fixture/formatter" GOIMPORTS="$fixture/formatter" format-file "$fixture/sample.go"
[ "$result" -eq 0 ]
expect_trace "formatter:-w $fixture/sample.go
formatter:-w -local github.com/gizzahub/gzh-cli $fixture/sample.go"

echo 'Make contract regression passed (9 scenarios).'
