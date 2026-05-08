"""Rollout SDK — Python client for the Rollout feature flag platform."""

from .client import RolloutClient
from .evaluator import Evaluator, FlagStore, hash_bucket
from .models import (
    Clause,
    EvalContext,
    EvalResult,
    ExposureEvent,
    Flag,
    FlagEnvironment,
    RolloutConfig,
    SSEEvent,
    TargetingRule,
    WeightedVariation,
)

__all__ = [
    "RolloutClient",
    "Evaluator",
    "FlagStore",
    "hash_bucket",
    "Clause",
    "EvalContext",
    "EvalResult",
    "ExposureEvent",
    "Flag",
    "FlagEnvironment",
    "RolloutConfig",
    "SSEEvent",
    "TargetingRule",
    "WeightedVariation",
]

__version__ = "1.0.0"
