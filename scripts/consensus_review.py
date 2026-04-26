#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional


REVIEW_SCHEMA: Dict[str, Any] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["summary", "good", "findings", "questions"],
    "properties": {
        "summary": {"type": "string"},
        "good": {"type": "array", "items": {"type": "string"}},
        "findings": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": [
                    "severity",
                    "title",
                    "file",
                    "line",
                    "why_it_matters",
                    "suggested_fix",
                    "confidence",
                ],
                "properties": {
                    "severity": {"type": "string", "enum": ["high", "medium", "low"]},
                    "title": {"type": "string"},
                    "file": {"type": "string"},
                    "line": {"type": ["integer", "null"]},
                    "why_it_matters": {"type": "string"},
                    "suggested_fix": {"type": "string"},
                    "confidence": {"type": "string", "enum": ["high", "medium", "low"]},
                },
            },
        },
        "questions": {"type": "array", "items": {"type": "string"}},
    },
}


CONSENSUS_SCHEMA: Dict[str, Any] = {
    "type": "object",
    "additionalProperties": False,
    "required": ["evaluations", "overall_recommendation"],
    "properties": {
        "evaluations": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["finding_id", "decision", "reason"],
                "properties": {
                    "finding_id": {"type": "string"},
                    "decision": {"type": "string", "enum": ["agree", "disagree", "unsure"]},
                    "reason": {"type": "string"},
                },
            },
        },
        "overall_recommendation": {"type": "string"},
    },
}


class WorkflowError(RuntimeError):
    pass


class WorkflowPaused(RuntimeError):
    def __init__(self, stage: str, reason: str, details: str = "") -> None:
        super().__init__(reason)
        self.stage = stage
        self.reason = reason
        self.details = details


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run a Codex + Claude consensus code review workflow."
    )
    target = parser.add_mutually_exclusive_group()
    target.add_argument(
        "--base",
        help="Review committed changes against the given base ref using `git diff <base>...HEAD`.",
    )
    target.add_argument(
        "--uncommitted",
        action="store_true",
        help="Review staged, unstaged, and untracked local changes. This is the default.",
    )
    parser.add_argument(
        "--path",
        action="append",
        default=[],
        help="Limit the review to a relative or absolute path. Repeatable.",
    )
    parser.add_argument(
        "--changed-under",
        action="append",
        default=[],
        help="Review changed files under this path one by one. Repeatable.",
    )
    parser.add_argument(
        "--resume-report",
        help="Resume a previous multi-file run from the given report directory.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="After consensus, ask Codex to apply only the mutually agreed findings.",
    )
    parser.add_argument(
        "--report-dir",
        help="Optional output directory. Defaults to .ai-consensus/<timestamp>/ under the repo root.",
    )
    parser.add_argument(
        "--codex-model",
        help="Optional Codex CLI model override for review and apply steps.",
    )
    parser.add_argument(
        "--claude-model",
        help="Optional Claude model override for review and consensus steps.",
    )
    parser.add_argument(
        "--max-findings",
        type=int,
        default=6,
        help="Maximum findings each reviewer should return.",
    )
    parser.add_argument(
        "--extra-instructions",
        default="",
        help="Optional extra review instructions appended to the shared prompt.",
    )
    return parser.parse_args()


def validate_args(args: argparse.Namespace) -> None:
    if args.resume_report:
        conflicting = (
            args.base
            or args.uncommitted
            or args.path
            or args.changed_under
            or args.report_dir
            or args.apply
            or args.max_findings != 6
            or bool(args.extra_instructions)
        )
        if conflicting:
            raise WorkflowError(
                "--resume-report cannot be combined with target selection flags, --apply, "
                "--report-dir, --max-findings, or --extra-instructions."
            )
        return

    if args.path and args.changed_under:
        raise WorkflowError("--path and --changed-under cannot be used together.")

    if not args.base and not args.uncommitted:
        args.uncommitted = True


_CLAUDE_CODE_ENV_KEYS = {"CLAUDECODE", "CLAUDE_CODE_SSE_PORT", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_EXECPATH"}


def run_command(
    cmd: List[str],
    cwd: Path,
    *,
    input_text: Optional[str] = None,
    check: bool = True,
    extra_env: Optional[Dict[str, str]] = None,
    strip_claude_env: bool = False,
) -> subprocess.CompletedProcess:
    env = os.environ.copy()
    if strip_claude_env:
        for key in _CLAUDE_CODE_ENV_KEYS:
            env.pop(key, None)
    if extra_env:
        env.update(extra_env)
    completed = subprocess.run(
        cmd,
        cwd=str(cwd),
        input=input_text,
        text=True,
        capture_output=True,
        env=env,
    )
    if check and completed.returncode != 0:
        raise WorkflowError(format_command_error(cmd, completed))
    return completed


def format_command_error(
    cmd: List[str], completed: subprocess.CompletedProcess
) -> str:
    parts = [
        f"Command failed with exit code {completed.returncode}:",
        " ".join(cmd),
    ]
    if completed.stdout.strip():
        parts.append("")
        parts.append("STDOUT:")
        parts.append(completed.stdout.strip())
    if completed.stderr.strip():
        parts.append("")
        parts.append("STDERR:")
        parts.append(completed.stderr.strip())
    return "\n".join(parts)


def ensure_command(name: str) -> str:
    path = shutil.which(name)
    if path is None:
        raise WorkflowError(f"Required command not found in PATH: {name}")
    return path


def write_text(path: Path, content: str) -> None:
    path.write_text(content, encoding="utf-8")


def write_json(path: Path, payload: Any) -> None:
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def read_json_if_exists(path: Path) -> Optional[Any]:
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise WorkflowError(f"Invalid JSON artifact: {path}") from exc


def read_text_if_exists(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8")


def write_status(
    report_dir: Path,
    *,
    status: str,
    stage: str,
    reason: str,
    details: str = "",
) -> None:
    payload = {
        "status": status,
        "stage": stage,
        "reason": reason,
        "details": details,
        "updated_at": datetime.now(timezone.utc).isoformat(),
    }
    write_json(report_dir / "status.json", payload)


def render_paused_summary(report_dir: Path, stage: str, reason: str) -> str:
    return "\n".join(
        [
            "# AI Consensus Review",
            "",
            f"Report directory: `{report_dir}`",
            "",
            "## Status",
            "",
            f"Paused at stage: `{stage}`",
            "",
            reason,
            "",
            "Refresh Claude authentication or budget, then rerun the workflow.",
            "",
        ]
    )


def render_failure_summary(report_dir: Path, reason: str) -> str:
    return "\n".join(
        [
            "# AI Consensus Review",
            "",
            f"Report directory: `{report_dir}`",
            "",
            "## Status",
            "",
            "Failed before completion.",
            "",
            reason,
            "",
        ]
    )


def is_claude_pause_condition(completed: subprocess.CompletedProcess) -> bool:
    text = (completed.stdout + "\n" + completed.stderr).lower()
    keywords = [
        "not logged in",
        "not authenticated",
        "authentication",
        "auth token",
        "token expired",
        "invalid api key",
        "api key",
        "run /login",
        "/login",
        "please run claude auth",
        "setup-token",
        "usage limit",
        "rate limit",
        "hit your limit",
        "your limit",
        "resets ",
        "quota",
        "credit balance",
        "insufficient credits",
        "billing",
        "subscription",
        "max_budget",
        "unauthorized",
        "forbidden",
    ]
    if any(keyword in text for keyword in keywords):
        return True
    # Also check structured JSON envelope from claude -p --output-format json
    try:
        envelope = json.loads(completed.stdout.strip())
        if isinstance(envelope, dict) and envelope.get("is_error"):
            result_text = str(envelope.get("result", "")).lower()
            return any(keyword in result_text for keyword in keywords)
    except (json.JSONDecodeError, ValueError):
        pass
    return False


def repo_root(cwd: Path) -> Path:
    completed = run_command(["git", "rev-parse", "--show-toplevel"], cwd)
    return Path(completed.stdout.strip()).resolve()


def normalize_paths(paths: List[str], root: Path) -> List[str]:
    normalized: List[str] = []
    root_resolved = root.resolve()
    for raw in paths:
        candidate = Path(raw).expanduser()
        if candidate.is_absolute():
            resolved = candidate.resolve()
            try:
                rel = resolved.relative_to(root_resolved)
            except ValueError as exc:
                raise WorkflowError(
                    f"Path is outside the repository root: {raw}"
                ) from exc
            normalized.append(str(rel))
            continue
        normalized.append(str(candidate))

    seen: Dict[str, None] = {}
    for item in normalized:
        seen[item] = None
    return list(seen.keys())


def git_diff_text(root: Path, args: argparse.Namespace, paths: List[str]) -> str:
    if args.base:
        cmd = ["git", "diff", "--no-ext-diff", "--unified=3", f"{args.base}...HEAD"]
        if paths:
            cmd.extend(["--", *paths])
        diff_text = run_command(cmd, root).stdout
        if not diff_text.strip():
            raise WorkflowError("No diff found for the selected base ref.")
        return diff_text

    sections: List[str] = []
    staged_cmd = ["git", "diff", "--cached", "--no-ext-diff", "--unified=3"]
    unstaged_cmd = ["git", "diff", "--no-ext-diff", "--unified=3"]
    if paths:
        staged_cmd.extend(["--", *paths])
        unstaged_cmd.extend(["--", *paths])

    staged = run_command(staged_cmd, root).stdout
    unstaged = run_command(unstaged_cmd, root).stdout
    if staged.strip():
        sections.append("## Staged changes\n\n" + staged.strip())
    if unstaged.strip():
        sections.append("## Unstaged changes\n\n" + unstaged.strip())

    untracked_cmd = ["git", "ls-files", "--others", "--exclude-standard"]
    if paths:
        untracked_cmd.extend(["--", *paths])
    untracked_output = run_command(untracked_cmd, root).stdout
    untracked_files = [line.strip() for line in untracked_output.splitlines() if line.strip()]
    if untracked_files:
        patches: List[str] = []
        for rel_path in untracked_files:
            target = root / rel_path
            completed = run_command(
                ["git", "diff", "--no-ext-diff", "--no-index", "--", os.devnull, str(target)],
                root,
                check=False,
            )
            if completed.returncode not in (0, 1):
                raise WorkflowError(
                    format_command_error(
                        ["git", "diff", "--no-ext-diff", "--no-index", "--", os.devnull, str(target)],
                        completed,
                    )
                )
            if completed.stdout.strip():
                patches.append(completed.stdout.strip())
        if patches:
            sections.append("## Untracked files\n\n" + "\n\n".join(patches))

    diff_text = "\n\n".join(sections).strip()
    if not diff_text:
        raise WorkflowError("No uncommitted changes found for the selected paths.")
    return diff_text


def changed_files_for_scope(
    root: Path,
    args: argparse.Namespace,
    scope: Optional[str],
) -> List[str]:
    files: Dict[str, None] = {}

    def add_output(cmd: List[str]) -> None:
        output = run_command(cmd, root).stdout
        for line in output.splitlines():
            rel_path = line.strip()
            if rel_path:
                files[rel_path] = None

    if args.base:
        cmd = ["git", "diff", "--name-only", f"{args.base}...HEAD"]
        if scope:
            cmd.extend(["--", scope])
        add_output(cmd)
    else:
        for cmd in (
            ["git", "diff", "--name-only", "--cached"],
            ["git", "diff", "--name-only"],
            ["git", "ls-files", "--others", "--exclude-standard"],
        ):
            if scope:
                cmd.extend(["--", scope])
            add_output(cmd)

    return sorted(files.keys())


def changed_files_for_prefixes(
    root: Path,
    args: argparse.Namespace,
    prefixes: List[str],
) -> List[str]:
    if not prefixes:
        return changed_files_for_scope(root, args, None)

    ordered: List[str] = []
    seen: Dict[str, None] = {}
    for prefix in prefixes:
        for rel_path in changed_files_for_scope(root, args, prefix):
            if rel_path in seen:
                continue
            seen[rel_path] = None
            ordered.append(rel_path)
    return ordered


def review_target_description(args: argparse.Namespace, paths: List[str]) -> str:
    scope = ", ".join(paths) if paths else "the whole repository"
    if args.base:
        return f"changes against base `{args.base}` scoped to {scope}"
    return f"uncommitted local changes scoped to {scope}"


def batch_target_description(args: argparse.Namespace, prefixes: List[str]) -> str:
    scope = ", ".join(prefixes) if prefixes else "the whole repository"
    if args.base:
        return f"changed files under {scope} against base `{args.base}`"
    return f"changed files under {scope} from the current working tree"


def read_skill_text(root: Path) -> str:
    skill_path = root / ".claude" / "skills" / "code-review" / "SKILL.md"
    if not skill_path.exists():
        return ""
    return skill_path.read_text(encoding="utf-8").strip()


def build_review_prompt(
    skill_text: str,
    target_desc: str,
    max_findings: int,
    extra_instructions: str,
) -> str:
    extra = extra_instructions.strip() or "None"
    return f"""You are one reviewer in a two-reviewer consensus workflow.

Review {target_desc}.

Only report findings important enough to justify a code change. Do not nitpick.
Use file paths relative to the repository root. Use 1-based line numbers when the diff makes them clear.
Return at most {max_findings} findings.
Return valid JSON only and match the schema exactly.

Shared review rubric:
{skill_text or "No project-specific rubric file was found."}

Additional instructions:
{extra}

The diff to review is provided below. For Codex runs it appears in the <stdin> block. For Claude runs it is embedded directly in the prompt.
"""


def build_consensus_prompt(
    skill_text: str,
    target_desc: str,
    extra_instructions: str,
) -> str:
    extra = extra_instructions.strip() or "None"
    return f"""You are the consensus step in a two-reviewer workflow.

Evaluate candidate findings for {target_desc}.

You will receive:
- the shared review rubric
- the git diff
- Codex's review result
- Claude's review result
- a combined candidate finding list

For each candidate finding_id, decide:
- agree: technically correct and important enough that the code should change now
- disagree: incorrect or not worth changing now
- unsure: plausible but not strong enough from the available context

Be independent and do not defer just because another reviewer raised it.
Return valid JSON only and match the schema exactly.

Shared review rubric:
{skill_text or "No project-specific rubric file was found."}

Additional instructions:
{extra}

For Codex runs the full payload is in the <stdin> block. For Claude runs it is embedded directly in the prompt.
"""


def build_apply_prompt(target_desc: str) -> str:
    return f"""You are applying mutually agreed review findings in the current repository.

Work from the repository state on disk and address only the agreed findings provided in the <stdin> block.
Do not make unrelated cleanup changes.
Preserve existing project conventions.
Run the most relevant tests you can after editing.

Scope:
{target_desc}

At the end, summarize:
- the files you changed
- which agreed findings you addressed
- what tests you ran
"""


def parse_json_payload(raw_text: str) -> Dict[str, Any]:
    text = raw_text.strip()
    if not text:
        raise WorkflowError("Model returned empty output when JSON was expected.")
    try:
        value = json.loads(text)
    except json.JSONDecodeError:
        first = text.find("{")
        last = text.rfind("}")
        if first == -1 or last == -1 or last <= first:
            raise WorkflowError("Could not parse model output as JSON.")
        value = json.loads(text[first : last + 1])

    if not isinstance(value, dict):
        raise WorkflowError("Expected a JSON object from the model.")
    return value


def normalize_review_payload(reviewer: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    findings: List[Dict[str, Any]] = []
    for index, finding in enumerate(payload.get("findings", []), start=1):
        if not isinstance(finding, dict):
            continue
        line_value = finding.get("line")
        normalized_line: Optional[int] = None
        if isinstance(line_value, int):
            normalized_line = line_value

        findings.append(
            {
                "finding_id": f"{reviewer}-{index}",
                "source": reviewer,
                "severity": str(finding.get("severity", "low")),
                "title": str(finding.get("title", "")).strip(),
                "file": str(finding.get("file", "")).strip(),
                "line": normalized_line,
                "why_it_matters": str(finding.get("why_it_matters", "")).strip(),
                "suggested_fix": str(finding.get("suggested_fix", "")).strip(),
                "confidence": str(finding.get("confidence", "medium")),
            }
        )

    return {
        "reviewer": reviewer,
        "summary": str(payload.get("summary", "")).strip(),
        "good": [str(item).strip() for item in payload.get("good", []) if str(item).strip()],
        "findings": findings,
        "questions": [
            str(item).strip()
            for item in payload.get("questions", [])
            if str(item).strip()
        ],
    }


def normalize_consensus_payload(payload: Dict[str, Any]) -> Dict[str, str]:
    decisions: Dict[str, str] = {}
    for item in payload.get("evaluations", []):
        if not isinstance(item, dict):
            continue
        finding_id = str(item.get("finding_id", "")).strip()
        decision = str(item.get("decision", "unsure")).strip()
        if not finding_id:
            continue
        decisions[finding_id] = decision
    return decisions


def invoke_codex_json(
    repo: Path,
    prompt: str,
    stdin_payload: str,
    schema: Dict[str, Any],
    report_dir: Path,
    output_name: str,
    *,
    model: Optional[str] = None,
    sandbox: str = "read-only",
) -> Dict[str, Any]:
    schema_file = report_dir / f"{output_name}_schema.json"
    write_json(schema_file, schema)
    write_text(report_dir / f"{output_name}_prompt.txt", prompt + "\n")
    write_text(report_dir / f"{output_name}_input.txt", stdin_payload + "\n")

    output_file = report_dir / f"{output_name}_last_message.json"
    cmd = [
        ensure_command("codex"),
        "exec",
        "--cd",
        str(repo),
        "--sandbox",
        sandbox,
        "--ephemeral",
        "--output-schema",
        str(schema_file),
        "-o",
        str(output_file),
    ]
    if model:
        cmd.extend(["--model", model])
    cmd.append(prompt)

    completed = run_command(cmd, repo, input_text=stdin_payload)
    write_text(report_dir / f"{output_name}_stdout.log", completed.stdout)
    write_text(report_dir / f"{output_name}_stderr.log", completed.stderr)

    if not output_file.exists():
        raise WorkflowError(f"Codex did not write the expected output file: {output_file}")
    return parse_json_payload(output_file.read_text(encoding="utf-8"))


def invoke_claude_json(
    repo: Path,
    prompt: str,
    schema: Dict[str, Any],
    report_dir: Path,
    output_name: str,
    *,
    model: Optional[str] = None,
    pause_stage: str = "claude",
) -> Dict[str, Any]:
    schema_arg = json.dumps(schema, separators=(",", ":"), ensure_ascii=False)
    write_text(report_dir / f"{output_name}_prompt.txt", prompt + "\n")
    cmd = [
        ensure_command("claude"),
        "-p",
        "--no-session-persistence",
        "--permission-mode",
        "dontAsk",
        "--tools",
        "",
        "--output-format",
        "json",
        "--json-schema",
        schema_arg,
    ]
    if model:
        cmd.extend(["--model", model])
    cmd.append(prompt)

    completed = run_command(cmd, repo, check=False, strip_claude_env=True)
    write_text(report_dir / f"{output_name}_stdout.log", completed.stdout)
    write_text(report_dir / f"{output_name}_stderr.log", completed.stderr)
    if completed.returncode != 0:
        error_text = format_command_error(cmd, completed)
        if is_claude_pause_condition(completed):
            raise WorkflowPaused(
                pause_stage,
                "Claude authentication or token/budget issue detected. Workflow paused before continuing.",
                error_text,
            )
        raise WorkflowError(error_text)

    # With --output-format json and --json-schema, the structured output is in
    # the "structured_output" field of the JSON envelope.
    envelope = parse_json_payload(completed.stdout)
    structured = envelope.get("structured_output")
    if not isinstance(structured, dict):
        raise WorkflowError(
            f"Claude response missing structured_output field. "
            f"is_error={envelope.get('is_error')}, result={envelope.get('result', '')[:200]}"
        )
    return structured


def invoke_codex_apply(
    repo: Path,
    prompt: str,
    stdin_payload: str,
    report_dir: Path,
    *,
    model: Optional[str] = None,
) -> str:
    output_file = report_dir / "codex_apply_last_message.txt"
    write_text(report_dir / "codex_apply_prompt.txt", prompt + "\n")
    write_text(report_dir / "codex_apply_input.txt", stdin_payload + "\n")
    cmd = [
        ensure_command("codex"),
        "exec",
        "--cd",
        str(repo),
        "--sandbox",
        "workspace-write",
        "--ephemeral",
        "-o",
        str(output_file),
    ]
    if model:
        cmd.extend(["--model", model])
    cmd.append(prompt)

    completed = run_command(cmd, repo, input_text=stdin_payload)
    write_text(report_dir / "codex_apply_stdout.log", completed.stdout)
    write_text(report_dir / "codex_apply_stderr.log", completed.stderr)
    if output_file.exists():
        return output_file.read_text(encoding="utf-8").strip()
    return ""


def build_consensus_input(
    diff_text: str,
    codex_review: Dict[str, Any],
    claude_review: Dict[str, Any],
) -> Dict[str, Any]:
    candidates = codex_review["findings"] + claude_review["findings"]
    return {
        "diff": diff_text,
        "codex_review": codex_review,
        "claude_review": claude_review,
        "candidate_findings": candidates,
    }


def agreed_findings(
    candidates: List[Dict[str, Any]],
    codex_decisions: Dict[str, str],
    claude_decisions: Dict[str, str],
) -> List[Dict[str, Any]]:
    agreed: List[Dict[str, Any]] = []
    for finding in candidates:
        finding_id = finding["finding_id"]
        codex_decision = codex_decisions.get(finding_id, "unsure")
        claude_decision = claude_decisions.get(finding_id, "unsure")
        if codex_decision == "agree" and claude_decision == "agree":
            agreed.append(finding)
    return agreed


def render_summary(
    report_dir: Path,
    target_desc: str,
    codex_review: Dict[str, Any],
    claude_review: Dict[str, Any],
    agreed: List[Dict[str, Any]],
    *,
    apply_message: str = "",
) -> str:
    lines = [
        "# AI Consensus Review",
        "",
        f"Report directory: `{report_dir}`",
        f"Target: {target_desc}",
        "",
        f"- Codex findings: {len(codex_review['findings'])}",
        f"- Claude findings: {len(claude_review['findings'])}",
        f"- Mutually agreed findings: {len(agreed)}",
        "",
        "## Agreed Findings",
    ]
    if not agreed:
        lines.append("")
        lines.append("No mutually agreed findings.")
    else:
        lines.append("")
        for finding in agreed:
            location = finding["file"] or "<unknown file>"
            if finding["line"] is not None:
                location = f"{location}:{finding['line']}"
            lines.append(
                f"- [{finding['severity']}] {location} - {finding['title']} ({finding['source']})"
            )
            lines.append(f"  Why: {finding['why_it_matters']}")
            lines.append(f"  Fix: {finding['suggested_fix']}")
            lines.append("")

    if apply_message:
        lines.append("## Apply Result")
        lines.append("")
        lines.append(apply_message)
        lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def create_report_dir(root: Path, requested: Optional[str]) -> Path:
    if requested:
        report_dir = Path(requested).expanduser()
        if not report_dir.is_absolute():
            report_dir = root / report_dir
    else:
        stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
        report_dir = root / ".ai-consensus" / stamp
    report_dir.mkdir(parents=True, exist_ok=True)
    return report_dir


def slugify(value: str) -> str:
    return re.sub(r"[^A-Za-z0-9]+", "-", value).strip("-").lower() or "item"


def run_single_workflow(
    root: Path,
    args: argparse.Namespace,
    report_dir: Path,
    skill_text: str,
    paths: List[str],
) -> Dict[str, Any]:
    report_dir.mkdir(parents=True, exist_ok=True)
    write_status(
        report_dir,
        status="running",
        stage="initializing",
        reason="workflow started",
    )

    target_desc = review_target_description(args, paths)
    saved_target = read_text_if_exists(report_dir / "target.txt").strip()
    if saved_target:
        target_desc = saved_target
    else:
        write_text(report_dir / "target.txt", target_desc + "\n")

    diff_text = read_text_if_exists(report_dir / "diff.patch")
    if not diff_text.strip():
        diff_text = git_diff_text(root, args, paths)
        write_text(report_dir / "diff.patch", diff_text)
    if skill_text:
        write_text(report_dir / "skill.md", skill_text + "\n")

    try:
        review_prompt = build_review_prompt(
            skill_text,
            target_desc,
            args.max_findings,
            args.extra_instructions,
        )

        codex_review_payload = read_json_if_exists(report_dir / "codex_review.json")
        if isinstance(codex_review_payload, dict):
            codex_review = codex_review_payload
        else:
            codex_raw = invoke_codex_json(
                root,
                review_prompt,
                diff_text,
                REVIEW_SCHEMA,
                report_dir,
                "codex_review",
                model=args.codex_model,
                sandbox="read-only",
            )
            codex_review = normalize_review_payload("codex", codex_raw)
            write_json(report_dir / "codex_review.json", codex_review)

        claude_review_payload = read_json_if_exists(report_dir / "claude_review.json")
        if isinstance(claude_review_payload, dict):
            claude_review = claude_review_payload
        else:
            claude_prompt = review_prompt + "\n\nDiff:\n\n" + diff_text
            claude_raw = invoke_claude_json(
                root,
                claude_prompt,
                REVIEW_SCHEMA,
                report_dir,
                "claude_review",
                model=args.claude_model,
                pause_stage="claude_review",
            )
            claude_review = normalize_review_payload("claude", claude_raw)
            write_json(report_dir / "claude_review.json", claude_review)

        consensus_input = build_consensus_input(diff_text, codex_review, claude_review)
        write_json(report_dir / "consensus_input.json", consensus_input)
        consensus_prompt = build_consensus_prompt(
            skill_text,
            target_desc,
            args.extra_instructions,
        )

        codex_consensus_raw = read_json_if_exists(report_dir / "codex_consensus.json")
        if not isinstance(codex_consensus_raw, dict):
            codex_consensus_raw = invoke_codex_json(
                root,
                consensus_prompt,
                json.dumps(consensus_input, indent=2, ensure_ascii=False),
                CONSENSUS_SCHEMA,
                report_dir,
                "codex_consensus",
                model=args.codex_model,
                sandbox="read-only",
            )
            write_json(report_dir / "codex_consensus.json", codex_consensus_raw)

        claude_consensus_raw = read_json_if_exists(report_dir / "claude_consensus.json")
        if not isinstance(claude_consensus_raw, dict):
            claude_consensus_prompt = (
                consensus_prompt
                + "\n\nConsensus payload:\n\n"
                + json.dumps(consensus_input, indent=2, ensure_ascii=False)
            )
            claude_consensus_raw = invoke_claude_json(
                root,
                claude_consensus_prompt,
                CONSENSUS_SCHEMA,
                report_dir,
                "claude_consensus",
                model=args.claude_model,
                pause_stage="claude_consensus",
            )
            write_json(report_dir / "claude_consensus.json", claude_consensus_raw)

        codex_decisions = normalize_consensus_payload(codex_consensus_raw)
        claude_decisions = normalize_consensus_payload(claude_consensus_raw)
        agreed = agreed_findings(
            consensus_input["candidate_findings"],
            codex_decisions,
            claude_decisions,
        )
        write_json(report_dir / "agreed_findings.json", agreed)

        apply_message = ""
        if args.apply and agreed:
            apply_message = read_text_if_exists(report_dir / "apply_summary.txt").strip()
            if not apply_message:
                apply_prompt = build_apply_prompt(target_desc)
                apply_input = {
                    "target": target_desc,
                    "agreed_findings": agreed,
                }
                write_json(report_dir / "codex_apply_input.json", apply_input)
                apply_message = invoke_codex_apply(
                    root,
                    apply_prompt,
                    json.dumps(apply_input, indent=2, ensure_ascii=False),
                    report_dir,
                    model=args.codex_model,
                )
                if apply_message:
                    write_text(report_dir / "apply_summary.txt", apply_message + "\n")

        summary = render_summary(
            report_dir,
            target_desc,
            codex_review,
            claude_review,
            agreed,
            apply_message=apply_message,
        )
        write_text(report_dir / "summary.md", summary)
        write_status(
            report_dir,
            status="completed",
            stage="done",
            reason="workflow completed successfully",
        )

        return {
            "target_desc": target_desc,
            "codex_findings": len(codex_review["findings"]),
            "claude_findings": len(claude_review["findings"]),
            "agreed_count": len(agreed),
            "summary_path": report_dir / "summary.md",
        }
    except WorkflowPaused as exc:
        write_status(
            report_dir,
            status="paused",
            stage=exc.stage,
            reason=exc.reason,
            details=exc.details,
        )
        paused_summary = render_paused_summary(report_dir, exc.stage, exc.reason)
        write_text(report_dir / "summary.md", paused_summary)
        raise
    except WorkflowError as exc:
        write_status(
            report_dir,
            status="failed",
            stage="workflow",
            reason="workflow failed",
            details=str(exc),
        )
        write_text(report_dir / "summary.md", render_failure_summary(report_dir, str(exc)))
        raise


def batch_state_file(report_dir: Path) -> Path:
    return report_dir / "batch_state.json"


def build_batch_state(
    args: argparse.Namespace,
    prefixes: List[str],
    files: List[str],
) -> Dict[str, Any]:
    items: List[Dict[str, Any]] = []
    for index, path in enumerate(files, start=1):
        items.append(
            {
                "path": path,
                "slug": f"{index:03d}-{slugify(path)}",
                "status": "pending",
                "attempts": 0,
                "codex_findings": 0,
                "claude_findings": 0,
                "agreed_findings": 0,
                "last_attempt_report_dir": "",
                "last_summary": "",
                "last_pause_reason": "",
                "last_error": "",
            }
        )

    return {
        "version": 1,
        "kind": "batch",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "updated_at": datetime.now(timezone.utc).isoformat(),
        "status": "pending",
        "target": {
            "base": args.base,
            "uncommitted": bool(args.uncommitted),
            "apply": bool(args.apply),
            "codex_model": args.codex_model,
            "claude_model": args.claude_model,
            "max_findings": args.max_findings,
            "extra_instructions": args.extra_instructions,
            "changed_under": prefixes,
        },
        "items": items,
        "current_index": None,
        "current_path": "",
        "paused_reason": "",
    }


def save_batch_state(report_dir: Path, state: Dict[str, Any]) -> None:
    state["updated_at"] = datetime.now(timezone.utc).isoformat()
    write_json(batch_state_file(report_dir), state)


def load_batch_state(report_dir: Path) -> Dict[str, Any]:
    path = batch_state_file(report_dir)
    if not path.exists():
        raise WorkflowError(f"Batch state file not found: {path}")
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise WorkflowError(f"Invalid batch state in: {path}") from exc
    if not isinstance(payload, dict):
        raise WorkflowError(f"Invalid batch state in: {path}")
    return payload


def build_batch_args_from_state(
    cli_args: argparse.Namespace,
    state: Dict[str, Any],
) -> argparse.Namespace:
    target = state.get("target", {})
    return argparse.Namespace(
        base=target.get("base"),
        uncommitted=target.get("uncommitted", False),
        path=[],
        changed_under=target.get("changed_under", []),
        resume_report=cli_args.resume_report,
        apply=target.get("apply", False),
        report_dir=None,
        codex_model=cli_args.codex_model or target.get("codex_model"),
        claude_model=cli_args.claude_model or target.get("claude_model"),
        max_findings=int(target.get("max_findings", 6)),
        extra_instructions=str(target.get("extra_instructions", "")),
    )


def render_batch_summary(report_dir: Path, state: Dict[str, Any]) -> str:
    total = len(state["items"])
    completed = sum(1 for item in state["items"] if item["status"] == "completed")
    paused = sum(1 for item in state["items"] if item["status"] == "paused")
    failed = sum(1 for item in state["items"] if item["status"] == "failed")
    lines = [
        "# AI Consensus Batch Review",
        "",
        f"Report directory: `{report_dir}`",
        f"Status: `{state['status']}`",
        "",
        f"- Total files: {total}",
        f"- Completed: {completed}",
        f"- Paused: {paused}",
        f"- Failed: {failed}",
        "",
    ]
    current_path = state.get("current_path", "")
    if current_path:
        lines.append(f"Current file: `{current_path}`")
        lines.append("")
    if state.get("paused_reason"):
        lines.append("## Pause Reason")
        lines.append("")
        lines.append(state["paused_reason"])
        lines.append("")
        lines.append(
            f"Resume with: `python3 scripts/consensus_review.py --resume-report {report_dir}`"
        )
        lines.append("")

    lines.append("## Files")
    lines.append("")
    for item in state["items"]:
        line = f"- [{item['status']}] `{item['path']}`"
        if item.get("attempts"):
            line += f" (attempts: {item['attempts']})"
        if item.get("agreed_findings"):
            line += f", agreed findings: {item['agreed_findings']}"
        lines.append(line)
        if item.get("last_summary"):
            lines.append(f"  Summary: `{item['last_summary']}`")
        if item.get("last_pause_reason"):
            lines.append(f"  Pause: {item['last_pause_reason']}")
        if item.get("last_error"):
            lines.append(f"  Error: {item['last_error']}")

    lines.append("")
    return "\n".join(lines)


def print_batch_status(report_dir: Path, state: Dict[str, Any]) -> None:
    total = len(state["items"])
    completed = sum(1 for item in state["items"] if item["status"] == "completed")
    print(f"Report directory: {report_dir}")
    print(f"Batch status: {state['status']}")
    print(f"Files completed: {completed}/{total}")
    if state.get("current_path"):
        print(f"Current file: {state['current_path']}")
    print(f"Summary: {report_dir / 'summary.md'}")


def run_batch_workflow(
    root: Path,
    args: argparse.Namespace,
    skill_text: str,
    report_dir: Path,
    state: Dict[str, Any],
) -> int:
    write_status(
        report_dir,
        status="running",
        stage="batch",
        reason="batch workflow running",
    )
    save_batch_state(report_dir, state)
    write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))

    files_root = report_dir / "files"
    files_root.mkdir(parents=True, exist_ok=True)

    for index, item in enumerate(state["items"]):
        if item["status"] == "completed":
            continue

        reuse_attempt = (
            item["status"] in {"paused", "running"}
            and bool(item.get("last_attempt_report_dir"))
            and (report_dir / str(item["last_attempt_report_dir"])).exists()
        )
        if reuse_attempt:
            attempt_number = int(item.get("attempts", 1))
            attempt_dir = report_dir / str(item["last_attempt_report_dir"])
        else:
            attempt_number = int(item.get("attempts", 0)) + 1
            item_root = files_root / item["slug"]
            attempt_dir = item_root / f"attempt-{attempt_number:03d}"
            attempt_dir.mkdir(parents=True, exist_ok=True)
            item["attempts"] = attempt_number
            item["last_attempt_report_dir"] = str(attempt_dir.relative_to(report_dir))

        item["status"] = "running"
        item["last_error"] = ""
        item["last_pause_reason"] = ""
        state["status"] = "running"
        state["current_index"] = index
        state["current_path"] = item["path"]
        state["paused_reason"] = ""
        save_batch_state(report_dir, state)
        write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))

        try:
            result = run_single_workflow(root, args, attempt_dir, skill_text, [item["path"]])
            item["status"] = "completed"
            item["codex_findings"] = result["codex_findings"]
            item["claude_findings"] = result["claude_findings"]
            item["agreed_findings"] = result["agreed_count"]
            item["last_summary"] = str((attempt_dir / "summary.md").relative_to(report_dir))
            save_batch_state(report_dir, state)
            write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))
        except WorkflowPaused as exc:
            item["status"] = "paused"
            item["last_pause_reason"] = exc.reason
            item["last_summary"] = str((attempt_dir / "summary.md").relative_to(report_dir))
            state["status"] = "paused"
            state["current_index"] = index
            state["current_path"] = item["path"]
            state["paused_reason"] = exc.reason
            save_batch_state(report_dir, state)
            write_status(
                report_dir,
                status="paused",
                stage=exc.stage,
                reason=exc.reason,
                details=exc.details,
            )
            write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))
            print_batch_status(report_dir, state)
            return 2
        except WorkflowError as exc:
            item["status"] = "failed"
            item["last_error"] = str(exc)
            if (attempt_dir / "summary.md").exists():
                item["last_summary"] = str((attempt_dir / "summary.md").relative_to(report_dir))
            state["status"] = "failed"
            state["current_index"] = index
            state["current_path"] = item["path"]
            save_batch_state(report_dir, state)
            write_status(
                report_dir,
                status="failed",
                stage="batch",
                reason="batch workflow failed",
                details=str(exc),
            )
            write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))
            raise

    state["status"] = "completed"
    state["current_index"] = None
    state["current_path"] = ""
    state["paused_reason"] = ""
    save_batch_state(report_dir, state)
    write_status(
        report_dir,
        status="completed",
        stage="done",
        reason="batch workflow completed successfully",
    )
    write_text(report_dir / "summary.md", render_batch_summary(report_dir, state))
    print_batch_status(report_dir, state)
    return 0


def run_single_mode(root: Path, args: argparse.Namespace, skill_text: str) -> int:
    report_dir = create_report_dir(root, args.report_dir)
    paths = normalize_paths(args.path, root)
    result = run_single_workflow(root, args, report_dir, skill_text, paths)
    print(f"Report directory: {report_dir}")
    print(f"Target: {result['target_desc']}")
    print(f"Codex findings: {result['codex_findings']}")
    print(f"Claude findings: {result['claude_findings']}")
    print(f"Mutually agreed findings: {result['agreed_count']}")
    if args.apply:
        if result["agreed_count"] > 0:
            print("Apply step: completed or attempted")
        else:
            print("Apply step: skipped because there were no mutually agreed findings")
    print(f"Summary: {report_dir / 'summary.md'}")
    return 0


def run_batch_mode(root: Path, args: argparse.Namespace, skill_text: str) -> int:
    prefixes = normalize_paths(args.changed_under, root)
    files = changed_files_for_prefixes(root, args, prefixes)
    if not files:
        raise WorkflowError("No changed files found under the selected prefixes.")

    report_dir = create_report_dir(root, args.report_dir)
    write_text(report_dir / "target.txt", batch_target_description(args, prefixes) + "\n")
    write_json(
        report_dir / "selected_files.json",
        {
            "prefixes": prefixes,
            "files": files,
        },
    )
    if skill_text:
        write_text(report_dir / "skill.md", skill_text + "\n")

    state = build_batch_state(args, prefixes, files)
    return run_batch_workflow(root, args, skill_text, report_dir, state)


def resume_batch_mode(root: Path, cli_args: argparse.Namespace, skill_text: str) -> int:
    report_dir = Path(cli_args.resume_report).expanduser()
    if not report_dir.is_absolute():
        report_dir = root / report_dir
    report_dir = report_dir.resolve()
    state = load_batch_state(report_dir)
    args = build_batch_args_from_state(cli_args, state)
    return run_batch_workflow(root, args, skill_text, report_dir, state)


def main() -> int:
    args = parse_args()
    validate_args(args)
    root = repo_root(Path.cwd())
    skill_text = read_skill_text(root)

    if args.resume_report:
        return resume_batch_mode(root, args, skill_text)
    if args.changed_under:
        return run_batch_mode(root, args, skill_text)
    return run_single_mode(root, args, skill_text)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except WorkflowPaused as exc:
        print(exc.reason, file=sys.stderr)
        sys.exit(2)
    except WorkflowError as exc:
        print(f"consensus_review.py: {exc}", file=sys.stderr)
        sys.exit(1)
