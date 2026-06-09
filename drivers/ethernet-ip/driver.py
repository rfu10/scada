"""EtherNet/IP (CIP) driver for the scada.io operator.

Uses pycomm3 (https://github.com/ottowayi/pycomm3) to talk to Allen-Bradley /
Rockwell ControlLogix, CompactLogix, and Micro8xx controllers.

Tag address format:  "Program:MainProgram.TagName"   (program-scoped)
                     "TagName"                        (controller-scoped)
                     "TagName[3]"                     (array element)
                     "TagName.SubMember"              (UDT member)
"""
from __future__ import annotations

import logging
from typing import Any

from pycomm3 import LogixDriver, RequestError, ResponseError

from scada_driver_sdk import (
    BaseDriver,
    ConnectResponse,
    HealthResponse,
    HealthStatus,
    TagAddress,
    TagValue,
    WriteResponse,
    run,
)

log = logging.getLogger(__name__)


class EtherNetIPDriver(BaseDriver):
    """EtherNet/IP CIP driver backed by pycomm3.LogixDriver."""

    def __init__(self) -> None:
        self._plc: LogixDriver | None = None
        self._endpoint: str = ""

    # ── Connection ────────────────────────────────────────────────────────────

    async def connect(self, endpoint: str, params: dict[str, str]) -> ConnectResponse:
        """Open a LogixDriver session.

        params keys:
          slot       — backplane slot number (default "0")
          micro800   — "true" for Micro8xx controllers
        """
        try:
            await self.disconnect()
            self._endpoint = endpoint

            kwargs: dict[str, Any] = {
                "slot": int(params.get("slot", "0")),
                "micro800": params.get("micro800", "").lower() == "true",
            }

            self._plc = LogixDriver(endpoint, **kwargs)
            self._plc.open()

            info = self._plc.info
            device_info = (
                f"{info.get('vendor', '')} {info.get('product_name', '')} "
                f"rev {info.get('revision', {}).get('major', '?')}."
                f"{info.get('revision', {}).get('minor', '?')}"
            ).strip()

            log.info("connected to %s (%s)", endpoint, device_info)
            return ConnectResponse(success=True, deviceInfo=device_info)

        except (RequestError, ResponseError, Exception) as exc:
            log.error("connect failed: %s", exc)
            self._plc = None
            return ConnectResponse(success=False, error=str(exc))

    async def disconnect(self) -> None:
        if self._plc is not None:
            try:
                self._plc.close()
            except Exception:
                pass
            self._plc = None

    # ── Read ─────────────────────────────────────────────────────────────────

    async def read(self, tags: list[TagAddress]) -> list[TagValue]:
        if self._plc is None:
            return [TagValue(address=t.address, quality="BAD") for t in tags]

        tag_names = [t.address for t in tags]
        try:
            # pycomm3 returns a single Tag namedtuple for one address,
            # or a list for multiple.
            results = self._plc.read(*tag_names)
            if not isinstance(results, list):
                results = [results]
        except (RequestError, ResponseError) as exc:
            log.warning("read error: %s", exc)
            return [TagValue(address=t.address, quality="BAD") for t in tags]

        output: list[TagValue] = []
        for tag_addr, result in zip(tags, results):
            if result is None or result.error:
                output.append(TagValue(address=tag_addr.address, quality="BAD"))
            else:
                output.append(
                    TagValue(
                        address=tag_addr.address,
                        value=_coerce(result.value),
                        quality="GOOD",
                    )
                )
        return output

    # ── Write ─────────────────────────────────────────────────────────────────

    async def write(self, values: list[TagValue]) -> WriteResponse:
        if self._plc is None:
            return WriteResponse(success=False, error="not connected")

        write_args = [(v.address, v.value) for v in values]
        try:
            results = self._plc.write(*write_args)
            if not isinstance(results, list):
                results = [results]
        except (RequestError, ResponseError) as exc:
            return WriteResponse(success=False, error=str(exc))

        errors = [r.error for r in results if r and r.error]
        if errors:
            return WriteResponse(success=False, error="; ".join(str(e) for e in errors))

        return WriteResponse(success=True)

    # ── Health ────────────────────────────────────────────────────────────────

    async def health(self) -> HealthResponse:
        if self._plc is None:
            return HealthResponse(status=HealthStatus.DISCONNECTED)
        try:
            # Use get_plc_name as a lightweight liveness ping.
            self._plc.get_plc_name()
            return HealthResponse(status=HealthStatus.CONNECTED)
        except Exception as exc:
            return HealthResponse(status=HealthStatus.ERROR, message=str(exc))


def _coerce(value: Any) -> Any:
    """Convert pycomm3 value types to JSON-serialisable primitives."""
    if isinstance(value, bool):
        return value
    if isinstance(value, (int, float, str)):
        return value
    # Structured / array types: return string representation.
    # Extend this for UDT support as needed.
    return str(value)


if __name__ == "__main__":
    run(EtherNetIPDriver())
