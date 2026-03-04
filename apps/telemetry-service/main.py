from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from datetime import datetime
from typing import Optional
import httpx
import uuid

app = FastAPI(title="Telemetry Service", version="1.0.0")

telemetry_store: list[dict] = []

SMART_HOME_URL = "http://app:8080"

class TelemetryRequest(BaseModel):
    device_id: str
    sensor_id: Optional[str] = None
    location: str
    metric_type: str = "TEMPERATURE"
    value: float
    unit: str = "°C"

class TelemetryRecord(BaseModel):
    id: str
    device_id: str
    sensor_id: Optional[str] = None
    location: str
    metric_type: str
    value: float
    unit: str
    recorded_at: datetime

@app.get("/health")
def health():
    return {"status": "ok", "service": "telemetry-service"}

@app.post("/telemetry", response_model=TelemetryRecord, status_code=201)
def receive_telemetry(data: TelemetryRequest):
    record = {
        "id": str(uuid.uuid4()),
        "device_id": data.device_id,
        "sensor_id": data.sensor_id,
        "location": data.location,
        "metric_type": data.metric_type,
        "value": data.value,
        "unit": data.unit,
        "recorded_at": datetime.utcnow(),
    }
    telemetry_store.append(record)

    # Интеграция с монолитом: обновить значение датчика если sensor_id указан
    if data.sensor_id:
        try:
            with httpx.Client(timeout=2.0) as client:
                client.patch(
                    f"{SMART_HOME_URL}/api/v1/sensors/{data.sensor_id}/value",
                    json={"value": data.value, "status": "active"},
                )
        except Exception:
            pass  # не блокируем запись телеметрии если монолит недоступен

    return record

@app.get("/telemetry", response_model=list[TelemetryRecord])
def get_telemetry(
    device_id: Optional[str] = None,
    location: Optional[str] = None,
    metric_type: Optional[str] = None,
    limit: int = 100,
):
    result = telemetry_store
    if device_id:
        result = [r for r in result if r["device_id"] == device_id]
    if location:
        result = [r for r in result if r["location"] == location]
    if metric_type:
        result = [r for r in result if r["metric_type"] == metric_type]
    return result[-limit:]

@app.get("/telemetry/{device_id}", response_model=list[TelemetryRecord])
def get_device_telemetry(device_id: str, limit: int = 50):
    result = [r for r in telemetry_store if r["device_id"] == device_id]
    if not result:
        raise HTTPException(status_code=404, detail="No telemetry found for this device")
    return result[-limit:]
