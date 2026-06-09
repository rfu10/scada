"""BaseDriver — the abstract class every Python SCADA driver must implement.

Subclass this, override the four abstract methods, then call:

    from scada_driver_sdk.server import run
    run(MyDriver())

The SDK's FastAPI server handles all HTTP routing, request parsing, and
Server-Sent Events; your driver only handles the industrial protocol.
"""
from __future__ import annotations

from abc import ABC, abstractmethod

from .models import (
    ConnectResponse,
    HealthResponse,
    HealthStatus,
    TagAddress,
    TagValue,
    WriteResponse,
)


class BaseDriver(ABC):
    """Abstract base for all SCADA protocol drivers."""

    # ── Required overrides ────────────────────────────────────────────────────

    @abstractmethod
    async def connect(self, endpoint: str, params: dict[str, str]) -> ConnectResponse:
        """Open a session with the field device at *endpoint*.

        Return ConnectResponse(success=True, device_info="…") on success,
        or ConnectResponse(success=False, error="…") on failure.
        """

    @abstractmethod
    async def disconnect(self) -> None:
        """Close the session cleanly."""

    @abstractmethod
    async def read(self, tags: list[TagAddress]) -> list[TagValue]:
        """Return the current value for each tag in *tags*.

        Maintain order; return TagValue(quality="BAD") for unreadable tags
        rather than raising an exception — let the operator decide severity.
        """

    @abstractmethod
    async def write(self, values: list[TagValue]) -> WriteResponse:
        """Write *values* to the device.

        Return WriteResponse(success=False, error="…") for partial failures.
        """

    # ── Optional override ────────────────────────────────────────────────────

    async def health(self) -> HealthResponse:
        """Return the current connection state.

        The default implementation returns CONNECTED if the driver object exists;
        override this with a real liveness check (e.g. read identity object).
        """
        return HealthResponse(status=HealthStatus.CONNECTED)
