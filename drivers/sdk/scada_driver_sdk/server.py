"""FastAPI server that wraps a BaseDriver and exposes the driver.v1 HTTP API.

Usage:
    from scada_driver_sdk.server import run
    run(MyDriver(), host="0.0.0.0", port=8080)

Or in Docker CMD:
    python -c "from mydriver import MyDriver; from scada_driver_sdk.server import run; run(MyDriver())"
"""
from __future__ import annotations

import asyncio
import json
import logging

from fastapi import FastAPI
from fastapi.responses import StreamingResponse
import uvicorn

from .base import BaseDriver
from .models import (
    ConnectRequest,
    ConnectResponse,
    DisconnectResponse,
    HealthResponse,
    ReadRequest,
    ReadResponse,
    TagAddress,
    WriteRequest,
    WriteResponse,
)

log = logging.getLogger(__name__)


def create_app(driver: BaseDriver) -> FastAPI:
    """Build and return the FastAPI application for *driver*."""
    app = FastAPI(
        title="SCADA Driver",
        description="driver.v1 HTTP interface — see proto/driver/v1/driver.proto",
        version="v1alpha1",
    )

    @app.post("/api/v1/connect", response_model=ConnectResponse)
    async def connect(req: ConnectRequest) -> ConnectResponse:
        return await driver.connect(req.endpoint, req.params)

    @app.post("/api/v1/disconnect", response_model=DisconnectResponse)
    async def disconnect() -> DisconnectResponse:
        await driver.disconnect()
        return DisconnectResponse()

    @app.post("/api/v1/read", response_model=ReadResponse)
    async def read(req: ReadRequest) -> ReadResponse:
        try:
            values = await driver.read(req.tags)
            return ReadResponse(values=values)
        except Exception as exc:
            log.exception("read error")
            return ReadResponse(values=[], error=str(exc))

    @app.post("/api/v1/write", response_model=WriteResponse)
    async def write(req: WriteRequest) -> WriteResponse:
        try:
            return await driver.write(req.values)
        except Exception as exc:
            log.exception("write error")
            return WriteResponse(success=False, error=str(exc))

    @app.get("/api/v1/health", response_model=HealthResponse)
    async def health() -> HealthResponse:
        return await driver.health()

    @app.get("/api/v1/subscribe")
    async def subscribe(tags: str, interval_ms: int = 1000) -> StreamingResponse:
        """Server-Sent Events stream of TagValue JSON objects.

        tags — JSON-encoded list of {"address": "…", "dataType": "…"}
        """
        tag_list = [TagAddress(**t) for t in json.loads(tags)]
        interval_s = max(interval_ms, 50) / 1000.0

        async def event_stream():
            while True:
                try:
                    values = await driver.read(tag_list)
                    for v in values:
                        yield f"data: {v.model_dump_json()}\n\n"
                except Exception as exc:
                    log.warning("subscribe read error: %s", exc)
                    yield f"data: {json.dumps({'error': str(exc)})}\n\n"
                await asyncio.sleep(interval_s)

        return StreamingResponse(event_stream(), media_type="text/event-stream")

    return app


def run(driver: BaseDriver, host: str = "0.0.0.0", port: int = 8080, **kwargs) -> None:
    """Start the uvicorn server for *driver*.  Blocks until shutdown."""
    logging.basicConfig(level=logging.INFO)
    app = create_app(driver)
    uvicorn.run(app, host=host, port=port, **kwargs)
