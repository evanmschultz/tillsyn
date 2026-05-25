#!/usr/bin/env bash
# gate_test.sh — empirical test battery for ta_action_gate.py (the PreToolUse
# gate hook). Pipes crafted PreToolUse JSON through the real hook and asserts
# the allow/deny decision. This is the logic tillsyn ports to the Go `till gate`
# subcommand — proving it here is the "truly tested" baseline before translation.
#
# Run: bash _sandbox_poc_reference/gate_test.sh
set -uo pipefail

HOOK="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/ta_action_gate.py"
PASS=0
FAIL=0

# run <name> <expected: allow|deny|defer> <env-allowlist-json | -> <stdin-json>
run() {
  local name="$1" expect="$2" allow="$3" input="$4" out got
  if [[ "$allow" == "-" ]]; then
    out="$(printf '%s' "$input" | python3 -B "$HOOK" 2>/dev/null)"
  else
    out="$(printf '%s' "$input" | TA_GATE_ALLOWLIST="$allow" python3 -B "$HOOK" 2>/dev/null)"
  fi
  if [[ -z "$out" ]]; then
    got="defer"
  elif printf '%s' "$out" | grep -q '"permissionDecision": "deny"'; then
    got="deny"
  elif printf '%s' "$out" | grep -q '"permissionDecision": "allow"'; then
    got="allow"
  else
    got="?($out)"
  fi
  if [[ "$got" == "$expect" ]]; then
    printf 'PASS  %-46s -> %s\n' "$name" "$got"
    PASS=$((PASS+1))
  else
    printf 'FAIL  %-46s -> got=%s want=%s\n' "$name" "$got" "$expect"
    FAIL=$((FAIL+1))
  fi
}

ALLOW_ONE='{"edit":["/repo/a.go"],"bash_deny":["git commit","git push","git add","mage install","go get"]}'
QA_NONE='{"edit":[],"bash_deny":["git commit","git push"]}'

# 1. orchestrator (no agent_id, no env) -> defer (dev keeps control)
run "orch defer (no agent_id)" defer - \
  '{"tool_name":"Edit","tool_input":{"file_path":"/repo/anything.go"},"cwd":"/repo"}'

# 2. in-scope edit -> allow
run "edit in-scope" allow "$ALLOW_ONE" \
  '{"tool_name":"Edit","tool_input":{"file_path":"/repo/a.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 3. off-scope edit -> deny
run "edit off-scope" deny "$ALLOW_ONE" \
  '{"tool_name":"Write","tool_input":{"file_path":"/repo/b.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 4. QA edit:[] -> deny all edits
run "QA edit:[] deny" deny "$QA_NONE" \
  '{"tool_name":"Edit","tool_input":{"file_path":"/repo/review.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-build-qa-proof"}'

# 5. git commit -> deny
run "bash git commit" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"git commit -m wip"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 6. git -C <dir> commit (global-flag evasion) -> deny
run "bash git -C dir commit (evasion)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"git -C /repo commit -m wip"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 7. env-prefixed path-prefixed git (FOO=1 /usr/bin/git push) -> deny
run "bash FOO=1 /usr/bin/git push (evasion)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"FOO=1 /usr/bin/git push origin main"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 8. shell-write redirection bypass -> deny
run "bash echo > off-scope (write bypass)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"echo pwned > /repo/b.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 9. sed -i in-place edit bypass -> deny
run "bash sed -i (write bypass)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"sed -i s/a/b/ /repo/a.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 10. interpreter bypass (python3 -c) -> deny
run "bash python3 -c (write bypass)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"python3 -c \"open(0,1)\""},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 11. mage install (bash_deny) -> deny
run "bash mage install (deny pattern)" deny "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"mage install"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 12. read-only git diff -> allow
run "bash git diff (read, allowed)" allow "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"git diff --stat"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 13. mage ci (build, allowed) -> allow
run "bash mage ci (allowed)" allow "$ALLOW_ONE" \
  '{"tool_name":"Bash","tool_input":{"command":"mage ci"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

# 14. allowed Read tool -> allow
run "Read tool (allowed)" allow "$ALLOW_ONE" \
  '{"tool_name":"Read","tool_input":{"file_path":"/repo/whatever.go"},"cwd":"/repo","agent_id":"x","agent_type":"ta-go-builder"}'

echo "-----"
echo "PASS=$PASS FAIL=$FAIL"
[[ "$FAIL" -eq 0 ]]
