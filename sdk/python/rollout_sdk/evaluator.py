"""Local flag evaluation engine mirroring the Go implementation."""

from __future__ import annotations

import hashlib
import re
import struct
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple

from .models import (
    Clause,
    EvalContext,
    EvalResult,
    Flag,
    FlagEnvironment,
    RolloutConfig,
    TargetingRule,
)


# ─── Hash Bucket (mirrors Go hashBucket) ─────────────────────────────────────


def hash_bucket(key: str, seed: int) -> int:
    """
    Deterministic bucket in [0, 10000) for sticky bucketing.

    Mirrors the Go implementation:
      - Write seed as 8-byte little-endian uint64
      - Write key as UTF-8 bytes
      - SHA-256, take first 4 bytes as big-endian uint32, mod 10000
    """
    seed_bytes = struct.pack("<Q", seed & 0xFFFFFFFFFFFFFFFF)
    key_bytes = key.encode("utf-8")
    h = hashlib.sha256(seed_bytes + key_bytes).digest()
    val = struct.unpack(">I", h[:4])[0]
    return val % 10000


# ─── Operator Evaluation ─────────────────────────────────────────────────────

_regex_cache: Dict[str, re.Pattern[str]] = {}


def _evaluate_operator(op: str, attr_val: str, values: List[str]) -> bool:
    if op == "eq":
        return len(values) > 0 and attr_val == values[0]
    elif op == "neq":
        return len(values) > 0 and attr_val != values[0]
    elif op == "in":
        return attr_val in values
    elif op == "not_in":
        return attr_val not in values
    elif op == "contains":
        return len(values) > 0 and values[0] in attr_val
    elif op == "starts_with":
        return len(values) > 0 and attr_val.startswith(values[0])
    elif op == "ends_with":
        return len(values) > 0 and attr_val.endswith(values[0])
    elif op == "gt":
        return _compare_numeric(attr_val, values) > 0
    elif op == "lt":
        return _compare_numeric(attr_val, values) < 0
    elif op == "regex":
        if len(values) == 0:
            return False
        pattern = values[0]
        if pattern not in _regex_cache:
            try:
                _regex_cache[pattern] = re.compile(pattern)
            except re.error:
                return False
        return bool(_regex_cache[pattern].search(attr_val))
    else:
        return False


def _compare_numeric(attr_val: str, values: List[str]) -> int:
    if len(values) == 0:
        return 0
    try:
        a = float(attr_val)
    except (ValueError, TypeError):
        a = 0.0
    try:
        b = float(values[0])
    except (ValueError, TypeError):
        b = 0.0
    if a > b:
        return 1
    if a < b:
        return -1
    return 0


# ─── Attribute Resolution ────────────────────────────────────────────────────


def _get_attribute_value(attribute: str, ctx: EvalContext) -> str:
    if attribute == "key":
        return ctx.key
    val = ctx.attributes.get(attribute)
    if val is not None:
        return str(val)
    return ""


# ─── Clause & Rule Matching ──────────────────────────────────────────────────


def _matches_clause(clause: Clause, ctx: EvalContext) -> bool:
    attr_val = _get_attribute_value(clause.attribute, ctx)
    result = _evaluate_operator(clause.operator, attr_val, clause.values)
    return (not result) if clause.negate else result


def _matches_rule(rule: TargetingRule, ctx: EvalContext) -> bool:
    return all(_matches_clause(c, ctx) for c in rule.clauses)


# ─── Rollout Resolution ──────────────────────────────────────────────────────


def _rollout_variation(rc: RolloutConfig, ctx: EvalContext) -> Any:
    if not rc.variations:
        return None
    if len(rc.variations) == 1:
        return rc.variations[0].variation

    bucket_key = ctx.key
    if rc.bucket_by and rc.bucket_by in ctx.attributes:
        bucket_key = str(ctx.attributes[rc.bucket_by])

    bucket = hash_bucket(bucket_key, rc.seed)

    cumulative = 0
    for wv in rc.variations:
        cumulative += wv.weight
        if bucket < cumulative:
            return wv.variation

    return rc.variations[-1].variation


def _resolve_variation(rule: TargetingRule, ctx: EvalContext) -> Any:
    if rule.rollout is not None:
        return _rollout_variation(rule.rollout, ctx)
    return rule.variation


def _resolve_fallthrough(ft: RolloutConfig, ctx: EvalContext) -> Any:
    return _rollout_variation(ft, ctx)


# ─── Flag Store ───────────────────────────────────────────────────────────────


class FlagStore:
    """In-memory store for flag definitions and their environment configs."""

    def __init__(self) -> None:
        self.flags: Dict[str, Flag] = {}  # flag_key -> Flag
        self.flag_environments: Dict[str, FlagEnvironment] = {}  # flag_id -> FlagEnvironment

    def update(self, flags: List[Flag], flag_envs: List[FlagEnvironment]) -> None:
        self.flags.clear()
        for f in flags:
            self.flags[f.key] = f
        self.flag_environments.clear()
        for fe in flag_envs:
            self.flag_environments[fe.flag_id] = fe


# ─── Evaluator ────────────────────────────────────────────────────────────────


class Evaluator:
    """Local flag evaluation engine mirroring the Go Evaluator."""

    def __init__(self, store: FlagStore) -> None:
        self._store = store

    def evaluate(
        self, flag_key: str, ctx: EvalContext, default_value: Any = None
    ) -> EvalResult:
        """Evaluate a single flag for the given context."""
        now = datetime.now(timezone.utc).isoformat()

        flag = self._store.flags.get(flag_key)
        if flag is None:
            return EvalResult(
                flag_key=flag_key,
                value=default_value,
                reason="ERROR",
                version=0,
                timestamp=now,
            )

        flag_env = self._store.flag_environments.get(flag.id)
        if flag_env is None:
            return EvalResult(
                flag_key=flag_key,
                value=default_value,
                reason="ERROR",
                version=0,
                timestamp=now,
            )

        # Kill switch
        if flag.kill_switch:
            return EvalResult(
                flag_key=flag_key,
                value=flag_env.off_variation if flag_env.off_variation is not None else default_value,
                reason="KILL_SWITCH",
                version=flag_env.version,
                timestamp=now,
            )

        # Disabled
        if not flag.enabled or not flag_env.enabled:
            return EvalResult(
                flag_key=flag_key,
                value=flag_env.off_variation if flag_env.off_variation is not None else default_value,
                reason="OFF",
                version=flag_env.version,
                timestamp=now,
            )

        # Dependency checks
        if flag.depends_on:
            for dep in flag.depends_on:
                dep_result = self.evaluate(dep, ctx, None)
                if dep_result.reason in ("OFF", "ERROR"):
                    return EvalResult(
                        flag_key=flag_key,
                        value=flag_env.off_variation if flag_env.off_variation is not None else default_value,
                        reason="DEPENDENCY_FAILED",
                        version=flag_env.version,
                        timestamp=now,
                    )

        # Targeting rules
        for rule in flag_env.rules:
            if _matches_rule(rule, ctx):
                value = _resolve_variation(rule, ctx)
                return EvalResult(
                    flag_key=flag_key,
                    value=value,
                    reason="RULE_MATCH",
                    version=flag_env.version,
                    timestamp=now,
                )

        # Fallthrough
        value = _resolve_fallthrough(flag_env.fallthrough, ctx)
        return EvalResult(
            flag_key=flag_key,
            value=value,
            reason="FALLTHROUGH",
            version=flag_env.version,
            timestamp=now,
        )

    def evaluate_all(self, ctx: EvalContext) -> List[EvalResult]:
        """Evaluate all flags in the store for the given context."""
        results: List[EvalResult] = []
        for flag_key in self._store.flags:
            results.append(self.evaluate(flag_key, ctx))
        return results
