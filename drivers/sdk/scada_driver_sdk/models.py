"""Wire types that mirror the driver.v1 proto contract.

These match the JSON field names expected by the Go operator's internal/driver/client.go.
"""
from __future__ import annotations

from enum import str as StrEnum
from typing import Union
from pydantic import BaseModel, Field


class TagAddress(BaseModel):
    address: str
    dataType: str


class TagValue(BaseModel):
    address: str
    value: Union[float, int, bool, str, None] = None
    quality: str = "GOOD"       # GOOD | BAD | UNCERTAIN
    timestampNs: int = 0        # Unix nanoseconds; 0 = not set


class ConnectRequest(BaseModel):
    endpoint: str
    params: dict[str, str] = Field(default_factory=dict)


class ConnectResponse(BaseModel):
    success: bool
    error: str = ""
    deviceInfo: str = ""


class DisconnectResponse(BaseModel):
    pass


class ReadRequest(BaseModel):
    tags: list[TagAddress]


class ReadResponse(BaseModel):
    values: list[TagValue]
    error: str = ""


class WriteRequest(BaseModel):
    values: list[TagValue]


class WriteResponse(BaseModel):
    success: bool
    error: str = ""


class HealthStatus(str):
    CONNECTED    = "CONNECTED"
    DISCONNECTED = "DISCONNECTED"
    ERROR        = "ERROR"
    UNKNOWN      = "UNKNOWN"


class HealthResponse(BaseModel):
    status: str = HealthStatus.UNKNOWN
    message: str = ""
