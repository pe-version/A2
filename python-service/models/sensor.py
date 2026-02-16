"""Pydantic models for sensor data."""

from datetime import datetime
from enum import Enum
from typing import Optional, Union

from pydantic import BaseModel, Field


class SensorType(str, Enum):
    """Valid sensor types."""

    TEMPERATURE = "temperature"
    MOTION = "motion"
    HUMIDITY = "humidity"
    LIGHT = "light"
    AIR_QUALITY = "air_quality"
    CO2 = "co2"
    CONTACT = "contact"
    PRESSURE = "pressure"


class SensorStatus(str, Enum):
    """Valid sensor statuses."""

    ACTIVE = "active"
    INACTIVE = "inactive"
    ERROR = "error"


class SensorBase(BaseModel):
    """Base sensor model with common fields."""

    name: str = Field(..., min_length=1, max_length=100, description="Sensor display name")
    type: SensorType = Field(..., description="Type of sensor")
    location: str = Field(..., min_length=1, max_length=100, description="Physical location")
    value: Union[float, int, bool] = Field(..., description="Current sensor value")
    unit: str = Field(..., min_length=1, max_length=50, description="Unit of measurement")
    status: SensorStatus = Field(..., description="Operational status")


class SensorCreate(SensorBase):
    """Model for creating a new sensor."""

    pass


class SensorUpdate(BaseModel):
    """Model for updating an existing sensor. All fields optional."""

    name: Optional[str] = Field(None, min_length=1, max_length=100)
    type: Optional[SensorType] = None
    location: Optional[str] = Field(None, min_length=1, max_length=100)
    value: Optional[Union[float, int, bool]] = None
    unit: Optional[str] = Field(None, min_length=1, max_length=50)
    status: Optional[SensorStatus] = None


class Sensor(SensorBase):
    """Complete sensor model with ID and timestamps."""

    id: str = Field(..., description="Unique sensor identifier")
    last_reading: str = Field(..., description="ISO 8601 timestamp of last reading")
    created_at: Optional[str] = Field(None, description="Creation timestamp")
    updated_at: Optional[str] = Field(None, description="Last update timestamp")

    class Config:
        from_attributes = True


class SensorList(BaseModel):
    """Response model for list of sensors."""

    sensors: list[Sensor]
    count: int
