"""Rollout feature flag client for Python."""

from __future__ import annotations

import json
import logging
import threading
import time
from datetime import datetime, timezone
from typing import Any, Callable, Dict, List, Optional, Set, TypeVar

import requests

from .evaluator import Evaluator, FlagStore
from .models import (
    EvalContext,
    EvalResult,
    ExposureEvent,
    Flag,
    FlagEnvironment,
    SSEEvent,
)

logger = logging.getLogger("rollout_sdk")

T = TypeVar("T")

# ─── Event Callbacks ──────────────────────────────────────────────────────────

FlagChangedCallback = Callable[[SSEEvent], None]
ReadyCallback = Callable[[], None]
ErrorCallback = Callable[[Exception], None]


# ─── RolloutClient ────────────────────────────────────────────────────────────


class RolloutClient:
    """
    Client SDK for the Rollout feature flag platform.

    Fetches and caches flag rules locally, supports SSE streaming for real-time
    updates with automatic polling fallback, and batches exposure events.

    Usage::

        client = RolloutClient(
            api_key="your-api-key",
            base_url="https://flags.example.com",
            environment="production",
        )
        client.wait_until_ready(timeout=5.0)

        enabled = client.get_boolean_value(
            "new-feature",
            EvalContext(key="user-123", attributes={"plan": "pro"}),
            default=False,
        )

        client.close()

    Or as a context manager::

        with RolloutClient(api_key=..., base_url=..., environment=...) as client:
            client.wait_until_ready()
            value = client.get_boolean_value("flag", ctx, False)
    """

    def __init__(
        self,
        api_key: str,
        base_url: str,
        environment: str,
        polling_interval: float = 30.0,
        enable_streaming: bool = True,
        flush_interval: float = 10.0,
        max_batch_size: int = 100,
    ) -> None:
        self._api_key = api_key
        self._base_url = base_url.rstrip("/")
        self._environment = environment
        self._polling_interval = polling_interval
        self._enable_streaming = enable_streaming
        self._flush_interval = flush_interval
        self._max_batch_size = max_batch_size

        self._store = FlagStore()
        self._evaluator = Evaluator(self._store)

        self._exposure_buffer: List[ExposureEvent] = []
        self._exposure_lock = threading.Lock()
        self._seen_exposures: Set[str] = set()

        self._listeners: Dict[str, List[Callable[..., Any]]] = {
            "flagChanged": [],
            "ready": [],
            "error": [],
        }

        self._ready_event = threading.Event()
        self._closed = False
        self._session = requests.Session()
        self._session.headers.update(
            {
                "Authorization": f"Bearer {self._api_key}",
                "Content-Type": "application/json",
            }
        )

        self._polling_thread: Optional[threading.Thread] = None
        self._streaming_thread: Optional[threading.Thread] = None
        self._flush_thread: Optional[threading.Thread] = None
        self._stop_event = threading.Event()

        # Start background initialisation
        self._init_thread = threading.Thread(target=self._init, daemon=True)
        self._init_thread.start()

    # ─── Context Manager ────────────────────────────────────────────────────

    def __enter__(self) -> "RolloutClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    # ─── Initialisation ─────────────────────────────────────────────────────

    def _init(self) -> None:
        try:
            self._fetch_flag_rules()
            self._ready_event.set()
            self._emit("ready")

            if self._enable_streaming:
                self._start_streaming()
            else:
                self._start_polling()

            self._start_flush_timer()
        except Exception as exc:
            logger.error("Rollout SDK init failed: %s", exc)
            self._emit("error", exc)
            # Still start polling so we can recover
            self._start_polling()
            self._start_flush_timer()

    def wait_until_ready(self, timeout: Optional[float] = None) -> bool:
        """Block until initial flag rules are loaded. Returns True if ready."""
        return self._ready_event.wait(timeout=timeout)

    def is_ready(self) -> bool:
        """Return True if the initial flag rules have been fetched."""
        return self._ready_event.is_set()

    # ─── Evaluation API ─────────────────────────────────────────────────────

    def evaluate(
        self,
        flag_key: str,
        context: EvalContext,
        default_value: Any = None,
    ) -> EvalResult:
        """Evaluate a flag locally using cached rules."""
        result = self._evaluator.evaluate(flag_key, context, default_value)
        self._track_exposure(flag_key, context, result)
        return result

    def get_boolean_value(
        self,
        flag_key: str,
        context: EvalContext,
        default: bool = False,
    ) -> bool:
        """Evaluate a flag and return a boolean."""
        result = self.evaluate(flag_key, context, default)
        return result.value if isinstance(result.value, bool) else default

    def get_string_value(
        self,
        flag_key: str,
        context: EvalContext,
        default: str = "",
    ) -> str:
        """Evaluate a flag and return a string."""
        result = self.evaluate(flag_key, context, default)
        return result.value if isinstance(result.value, str) else default

    def get_number_value(
        self,
        flag_key: str,
        context: EvalContext,
        default: float = 0.0,
    ) -> float:
        """Evaluate a flag and return a number."""
        result = self.evaluate(flag_key, context, default)
        return result.value if isinstance(result.value, (int, float)) else default

    def get_json_value(
        self,
        flag_key: str,
        context: EvalContext,
        default: Any = None,
    ) -> Any:
        """Evaluate a flag and return an arbitrary JSON value."""
        result = self.evaluate(flag_key, context, default)
        return result.value

    def get_all_flags(self, context: EvalContext) -> List[EvalResult]:
        """Evaluate all cached flags for the given context."""
        results = self._evaluator.evaluate_all(context)
        for result in results:
            self._track_exposure(result.flag_key, context, result)
        return results

    # ─── Event Emitter ──────────────────────────────────────────────────────

    def on(self, event: str, callback: Callable[..., Any]) -> "RolloutClient":
        """Register an event listener. Events: flagChanged, ready, error."""
        if event in self._listeners:
            self._listeners[event].append(callback)
        return self

    def off(self, event: str, callback: Callable[..., Any]) -> "RolloutClient":
        """Remove an event listener."""
        if event in self._listeners:
            try:
                self._listeners[event].remove(callback)
            except ValueError:
                pass
        return self

    def _emit(self, event: str, *args: Any) -> None:
        for cb in self._listeners.get(event, []):
            try:
                cb(*args)
            except Exception:
                pass  # Listener errors must not crash the SDK.

    # ─── Flag Fetching ──────────────────────────────────────────────────────

    def _fetch_flag_rules(self) -> None:
        url = f"{self._base_url}/v1/sdk/flags"
        params = {"environment": self._environment}

        resp = self._session.get(url, params=params, timeout=10)
        resp.raise_for_status()
        data = resp.json()

        flags = [Flag.from_dict(f) for f in data.get("flags", [])]
        flag_envs = [FlagEnvironment.from_dict(fe) for fe in data.get("flag_environments", [])]
        self._store.update(flags, flag_envs)

    # ─── SSE Streaming ──────────────────────────────────────────────────────

    def _start_streaming(self) -> None:
        if self._closed:
            return
        self._streaming_thread = threading.Thread(
            target=self._streaming_loop, daemon=True
        )
        self._streaming_thread.start()

    def _streaming_loop(self) -> None:
        try:
            import sseclient
        except ImportError:
            logger.warning("sseclient-py not installed, falling back to polling")
            self._start_polling()
            return

        url = f"{self._base_url}/v1/stream"
        params = {"env_id": self._environment}

        while not self._stop_event.is_set():
            try:
                resp = self._session.get(
                    url,
                    params=params,
                    stream=True,
                    timeout=(5, None),  # 5s connect timeout, no read timeout
                )
                resp.raise_for_status()

                client = sseclient.SSEClient(resp)
                for event in client.events():
                    if self._stop_event.is_set():
                        break

                    if event.event in (
                        "flag.updated",
                        "flag.created",
                        "flag.deleted",
                        "flag.toggled",
                    ):
                        try:
                            sse_event = SSEEvent.from_dict(json.loads(event.data))
                            self._handle_sse_event(sse_event)
                        except (json.JSONDecodeError, Exception):
                            pass

            except Exception as exc:
                if self._stop_event.is_set():
                    break
                logger.debug("SSE connection lost (%s), reconnecting in 5s", exc)
                self._stop_event.wait(5)

        # If SSE dies permanently, fall back to polling
        if not self._stop_event.is_set():
            self._start_polling()

    def _handle_sse_event(self, event: SSEEvent) -> None:
        try:
            self._fetch_flag_rules()
        except Exception:
            pass
        self._emit("flagChanged", event)

    # ─── Polling Fallback ────────────────────────────────────────────────────

    def _start_polling(self) -> None:
        if self._closed or self._polling_thread is not None:
            return
        self._polling_thread = threading.Thread(
            target=self._polling_loop, daemon=True
        )
        self._polling_thread.start()

    def _polling_loop(self) -> None:
        while not self._stop_event.wait(self._polling_interval):
            try:
                self._fetch_flag_rules()
            except Exception as exc:
                logger.debug("Polling fetch failed: %s", exc)

    # ─── Exposure Tracking ───────────────────────────────────────────────────

    def _track_exposure(
        self, flag_key: str, context: EvalContext, result: EvalResult
    ) -> None:
        dedupe_key = f"{flag_key}:{context.key}"
        if dedupe_key in self._seen_exposures:
            return
        self._seen_exposures.add(dedupe_key)

        event = ExposureEvent(
            flag_key=flag_key,
            environment_id=self._environment,
            user_key=context.key,
            variation=result.value,
            reason=result.reason,
            timestamp=datetime.now(timezone.utc).isoformat(),
        )

        with self._exposure_lock:
            self._exposure_buffer.append(event)
            if len(self._exposure_buffer) >= self._max_batch_size:
                self._flush_exposures_locked()

    def _start_flush_timer(self) -> None:
        if self._closed:
            return
        self._flush_thread = threading.Thread(
            target=self._flush_loop, daemon=True
        )
        self._flush_thread.start()

    def _flush_loop(self) -> None:
        while not self._stop_event.wait(self._flush_interval):
            self.flush_exposures()

    def flush_exposures(self) -> None:
        """Flush any pending exposure events to the server."""
        with self._exposure_lock:
            self._flush_exposures_locked()

    def _flush_exposures_locked(self) -> None:
        """Must be called while holding _exposure_lock."""
        if not self._exposure_buffer:
            return

        batch = list(self._exposure_buffer)
        self._exposure_buffer.clear()

        try:
            url = f"{self._base_url}/v1/sdk/exposures"
            self._session.post(
                url,
                json={"events": [e.to_dict() for e in batch]},
                timeout=5,
            )
        except Exception:
            # On failure, put events back for retry
            self._exposure_buffer = batch + self._exposure_buffer

    # ─── Cleanup ─────────────────────────────────────────────────────────────

    def close(self) -> None:
        """Flush pending exposures and shut down all background tasks."""
        if self._closed:
            return
        self._closed = True
        self._stop_event.set()

        # Best-effort final flush
        try:
            self.flush_exposures()
        except Exception:
            pass

        self._session.close()
        self._listeners.clear()
