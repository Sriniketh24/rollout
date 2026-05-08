"""Data models for the Rollout SDK, mirroring the Go internal/models package."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


# ─── Flag Types ───────────────────────────────────────────────────────────────


@dataclass
class WeightedVariation:
    variation: Any
    weight: int  # basis points 0-10000


@dataclass
class RolloutConfig:
    variations: List[WeightedVariation] = field(default_factory=list)
    bucket_by: str = ""
    seed: int = 0

    @classmethod
    def from_dict(cls, d: dict) -> "RolloutConfig":
        variations = [
            WeightedVariation(variation=v.get("variation"), weight=v.get("weight", 0))
            for v in d.get("variations", [])
        ]
        return cls(
            variations=variations,
            bucket_by=d.get("bucket_by", ""),
            seed=d.get("seed", 0),
        )


@dataclass
class Clause:
    attribute: str
    operator: str  # eq, neq, contains, starts_with, ends_with, in, not_in, gt, lt, regex
    values: List[str] = field(default_factory=list)
    negate: bool = False

    @classmethod
    def from_dict(cls, d: dict) -> "Clause":
        return cls(
            attribute=d.get("attribute", ""),
            operator=d.get("operator", ""),
            values=d.get("values", []),
            negate=d.get("negate", False),
        )


@dataclass
class TargetingRule:
    id: str = ""
    clauses: List[Clause] = field(default_factory=list)
    variation: Any = None
    rollout: Optional[RolloutConfig] = None
    priority: int = 0

    @classmethod
    def from_dict(cls, d: dict) -> "TargetingRule":
        clauses = [Clause.from_dict(c) for c in d.get("clauses", [])]
        rollout = None
        if d.get("rollout"):
            rollout = RolloutConfig.from_dict(d["rollout"])
        return cls(
            id=d.get("id", ""),
            clauses=clauses,
            variation=d.get("variation"),
            rollout=rollout,
            priority=d.get("priority", 0),
        )


@dataclass
class FlagEnvironment:
    flag_id: str = ""
    environment_id: str = ""
    enabled: bool = False
    rules: List[TargetingRule] = field(default_factory=list)
    fallthrough: RolloutConfig = field(default_factory=RolloutConfig)
    off_variation: Any = None
    version: int = 0
    updated_at: str = ""

    @classmethod
    def from_dict(cls, d: dict) -> "FlagEnvironment":
        rules = [TargetingRule.from_dict(r) for r in d.get("rules", [])]
        fallthrough = RolloutConfig.from_dict(d.get("fallthrough", {}))
        return cls(
            flag_id=d.get("flag_id", ""),
            environment_id=d.get("environment_id", ""),
            enabled=d.get("enabled", False),
            rules=rules,
            fallthrough=fallthrough,
            off_variation=d.get("off_variation"),
            version=d.get("version", 0),
            updated_at=d.get("updated_at", ""),
        )


@dataclass
class Flag:
    id: str = ""
    project_id: str = ""
    key: str = ""
    name: str = ""
    description: str = ""
    type: str = "boolean"  # boolean, string, number, json
    default_value: Any = None
    enabled: bool = False
    tags: List[str] = field(default_factory=list)
    depends_on: List[str] = field(default_factory=list)
    kill_switch: bool = False
    archived: bool = False
    created_by: str = ""
    created_at: str = ""
    updated_at: str = ""
    environments: List[FlagEnvironment] = field(default_factory=list)

    @classmethod
    def from_dict(cls, d: dict) -> "Flag":
        envs = [FlagEnvironment.from_dict(e) for e in d.get("environments", [])]
        return cls(
            id=d.get("id", ""),
            project_id=d.get("project_id", ""),
            key=d.get("key", ""),
            name=d.get("name", ""),
            description=d.get("description", ""),
            type=d.get("type", "boolean"),
            default_value=d.get("default_value"),
            enabled=d.get("enabled", False),
            tags=d.get("tags", []),
            depends_on=d.get("depends_on", []),
            kill_switch=d.get("kill_switch", False),
            archived=d.get("archived", False),
            created_by=d.get("created_by", ""),
            created_at=d.get("created_at", ""),
            updated_at=d.get("updated_at", ""),
            environments=envs,
        )


# ─── Evaluation ───────────────────────────────────────────────────────────────


@dataclass
class EvalContext:
    key: str
    anonymous: bool = False
    attributes: Dict[str, Any] = field(default_factory=dict)


@dataclass
class EvalResult:
    flag_key: str
    value: Any
    reason: str  # TARGET_MATCH, RULE_MATCH, FALLTHROUGH, OFF, KILL_SWITCH, ERROR, DEFAULT, DEPENDENCY_FAILED
    version: int = 0
    timestamp: str = ""


# ─── Exposure Events ─────────────────────────────────────────────────────────


@dataclass
class ExposureEvent:
    flag_key: str
    environment_id: str
    user_key: str
    variation: Any
    reason: str
    timestamp: str

    def to_dict(self) -> dict:
        return {
            "flag_key": self.flag_key,
            "environment_id": self.environment_id,
            "user_key": self.user_key,
            "variation": self.variation,
            "reason": self.reason,
            "timestamp": self.timestamp,
        }


# ─── SSE Events ──────────────────────────────────────────────────────────────


@dataclass
class SSEEvent:
    type: str  # flag.updated, flag.created, flag.deleted, flag.toggled, etc.
    flag_key: str = ""
    data: Any = None
    version: int = 0
    timestamp: str = ""

    @classmethod
    def from_dict(cls, d: dict) -> "SSEEvent":
        return cls(
            type=d.get("type", ""),
            flag_key=d.get("flag_key", ""),
            data=d.get("data"),
            version=d.get("version", 0),
            timestamp=d.get("timestamp", ""),
        )
