#!/bin/bash
# Exercise the actual repository recipes without installing or running tools.
set -euo pipefail

# Git configuration injection and redirects must not escape this test's fixture.
while IFS= read -r variable; do
    case "$variable" in GIT_*) unset "$variable" ;; esac
done < <(compgen -e)
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1

root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$root/tmp"
fixture=$(mktemp -d "$root/tmp/make-contract.XXXXXX")
cleanup() {
    local status=$?
    if [ "$status" -ne 0 ]; then
        cat "$fixture/stdout" "$fixture/stderr" >&2
    fi
    if [ -d "$fixture/repo" ]; then
        # Only this test-created repository is recursively removed.
        find "$fixture/repo" -print > "$fixture/cleanup-paths"
        rm -rf -- "$fixture/repo"
    fi
    rm -f -- "$fixture/override.mk" "$fixture/go" "$fixture/formatter" \
        "$fixture/trace" "$fixture/stdout" "$fixture/stderr" "$fixture/sample.go" \
        "$fixture/second.go" "$fixture/gofmt" "$fixture/diff.mk" "$fixture/cleanup-paths"
    rmdir -- "$fixture"
}
trap cleanup EXIT
export CONTRACT_TRACE="$fixture/trace"
export CONTRACT_FAIL_STAGE=""
export CONTRACT_FORMATTER_FAIL=""
scenarios=0

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
if [ -n "$CONTRACT_FORMATTER_FAIL" ] && [ "$*" = "$CONTRACT_FORMATTER_FAIL" ]; then
    echo "controlled-formatter-failure:$*" >&2
    exit 23
fi
EOF
chmod +x "$fixture/go" "$fixture/formatter"
export PATH="$fixture:$PATH"
printf 'package fixture\n' > "$fixture/sample.go"
printf 'package fixture\n' > "$fixture/second.go"

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
    scenarios=$((scenarios + 1))
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
expect_no_stdout() {
    if grep -Fq "$1" "$fixture/stdout"; then
        printf 'Unexpected success output: %s\n' "$1" >&2
        exit 1
    else
        local status=$?
        [ "$status" -eq 1 ] || exit "$status"
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

# Stop at the failing tool/file, preserving its stderr and withholding the
# failing file's success message and the final aggregate success message.
for file in "$fixture/sample.go" "$fixture/second.go"; do
    for tool in gofumpt goimports; do
        expected=""
        for current in "$fixture/sample.go" "$fixture/second.go"; do
            line="formatter:-w $current"
            expected="${expected}${expected:+$'\n'}$line"
            [ "$current" != "$file" ] || [ "$tool" != gofumpt ] || break
            line="formatter:-w -local github.com/gizzahub/gzh-cli $current"
            expected="${expected}${expected:+$'\n'}$line"
            [ "$current" != "$file" ] || break
        done
        args="-w $file"
        [ "$tool" != goimports ] || args="-w -local github.com/gizzahub/gzh-cli $file"
        export CONTRACT_FORMATTER_FAIL="$args"
        invoke GOFUMPT="$fixture/formatter" GOIMPORTS="$fixture/formatter" format-file "$fixture/sample.go" "$fixture/second.go"
        [ "$result" -ne 0 ]
        grep -Fq 'controlled-formatter-failure:' "$fixture/stderr"
        grep -Fq "$tool failed for $file" "$fixture/stderr"
        expect_no_stdout "File '$file' formatted successfully"
        expect_no_stdout 'All files processed'
        expect_trace "$expected"
    done
done
export CONTRACT_FORMATTER_FAIL=""

invoke format-file
[ "$result" -ne 0 ]
expect_trace ""
invoke GOFUMPT="$fixture/formatter" GOIMPORTS="$fixture/formatter" format-file "$fixture/missing.go"
[ "$result" -ne 0 ]
grep -Fq 'does not exist' "$fixture/stderr"
expect_trace ""

# Include the actual recipe in a test-owned repository: fmt-diff must never
# discover or format the author's working tree while testing itself.
mkdir "$fixture/repo"
git init --template= -q "$fixture/repo"
printf 'package fixture\n' > "$fixture/repo/first.go"
printf 'package fixture\n' > "$fixture/repo/second.go"
printf 'fixture\n' > "$fixture/repo/note.txt"
git -C "$fixture/repo" -c core.excludesFile=/dev/null add -- first.go second.go note.txt
git -C "$fixture/repo" -c core.hooksPath=/dev/null -c commit.gpgsign=false \
    -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm fixture
cat > "$fixture/diff.mk" <<EOF
include $root/.make/quality.mk
.PHONY: install-format-tools
install-format-tools:
	@:
EOF
invoke_diff() {
    scenarios=$((scenarios + 1))
    : > "$CONTRACT_TRACE"
    if make --no-print-directory -C "$fixture/repo" -f "$fixture/diff.mk" \
        GOFUMPT="$fixture/formatter" GOIMPORTS="$fixture/formatter" fmt-diff \
        > "$fixture/stdout" 2> "$fixture/stderr"; then
        result=0
    else
        result=$?
    fi
}
invoke_diff
[ "$result" -eq 0 ]
grep -Fq 'No Go files changed' "$fixture/stdout"
expect_trace ""
printf 'changed\n' >> "$fixture/repo/note.txt"
invoke_diff
[ "$result" -eq 0 ]
grep -Fq 'No Go files changed' "$fixture/stdout"
expect_trace ""
printf '// changed\n' >> "$fixture/repo/first.go"
printf '// changed\n' >> "$fixture/repo/second.go"
invoke_diff
[ "$result" -eq 0 ]
expect_trace $'formatter:-w first.go\nformatter:-w -local github.com/gizzahub/gzh-cli first.go\nformatter:-w second.go\nformatter:-w -local github.com/gizzahub/gzh-cli second.go'
for file in first.go second.go; do
    for tool in gofumpt goimports; do
        expected=""
        for current in first.go second.go; do
            expected="${expected}${expected:+$'\n'}formatter:-w $current"
            [ "$current" != "$file" ] || [ "$tool" != gofumpt ] || break
            expected="${expected}${expected:+$'\n'}formatter:-w -local github.com/gizzahub/gzh-cli $current"
            [ "$current" != "$file" ] || break
        done
        args="-w $file"
        [ "$tool" != goimports ] || args="-w -local github.com/gizzahub/gzh-cli $file"
        export CONTRACT_FORMATTER_FAIL="$args"
        invoke_diff
        [ "$result" -ne 0 ]
        grep -Fq 'controlled-formatter-failure:' "$fixture/stderr"
        grep -Fq "$tool failed for $file" "$fixture/stderr"
        expect_no_stdout 'Changed files formatted!'
        expect_trace "$expected"
    done
done
export CONTRACT_FORMATTER_FAIL=""
# An invalid revision query is not an empty successful change set.
invoke_diff_with_bad_head() {
    git -C "$fixture/repo" symbolic-ref HEAD refs/heads/unborn-contract-fixture
    invoke_diff
}
invoke_diff_with_bad_head
[ "$result" -ne 0 ]
expect_no_stdout 'No Go files changed'
expect_no_stdout 'Changed files formatted!'
expect_trace ""

# Exercise the actual read-only probes with a fixed executable, including
# failures which wrote partial stdout before returning an error.
cat > "$fixture/gofmt" <<'EOF'
#!/bin/sh
printf 'gofmt:%s\n' "$*" >> "$CONTRACT_TRACE"
case "$CONTRACT_GOFMT_MODE" in
    dirty|partial-error)
        if [ "$1" = -l ]; then
            printf 'first.go\n'
        else
            printf 'diff controlled-first.go\n'
        fi
        ;;
esac
case "$CONTRACT_GOFMT_MODE" in
    error|partial-error)
        printf 'controlled-gofmt-failure:%s\n' "$*" >&2
        exit 23
        ;;
esac
EOF
chmod +x "$fixture/gofmt"
invoke_probe() {
    scenarios=$((scenarios + 1))
    : > "$CONTRACT_TRACE"
    if make --no-print-directory -C "$fixture/repo" -f "$fixture/diff.mk" "$1" \
        > "$fixture/stdout" 2> "$fixture/stderr"; then
        result=0
    else
        result=$?
    fi
}
probe_before=$(cksum "$fixture/repo/first.go" "$fixture/repo/second.go" "$fixture/repo/note.txt")
for target in format-list format-diff; do
    flag=-l
    success='All files are properly formatted!'
    dirty='first.go'
    if [ "$target" = format-diff ]; then
        flag=-d
        success='No formatting differences found!'
        dirty='diff controlled-first.go'
    fi
    for mode in clean dirty error partial-error; do
        export CONTRACT_GOFMT_MODE="$mode"
        invoke_probe "$target"
        expect_trace "gofmt:$flag ."
        case "$mode" in
            clean)
                [ "$result" -eq 0 ]
                grep -Fq "$success" "$fixture/stdout"
                ;;
            dirty)
                [ "$result" -eq 0 ]
                grep -Fq "$dirty" "$fixture/stdout"
                expect_no_stdout "$success"
                ;;
            error|partial-error)
                [ "$result" -ne 0 ]
                grep -Fq "controlled-gofmt-failure:$flag ." "$fixture/stderr"
                grep -Fq 'Error 23' "$fixture/stderr"
                expect_no_stdout "$success"
                expect_no_stdout "$dirty"
                ;;
        esac
        probe_after=$(cksum "$fixture/repo/first.go" "$fixture/repo/second.go" "$fixture/repo/note.txt")
        [ "$probe_before" = "$probe_after" ]
    done
done

printf 'Make contract regression passed (%s scenarios).\n' "$scenarios"
