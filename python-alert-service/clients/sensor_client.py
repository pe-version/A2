"""Sensor service client with circuit breaker and retry logic."""

import logging

import httpx
import pybreaker
from tenacity import retry, stop_after_attempt, wait_exponential

logger = logging.getLogger("alert_service")


class SensorClient:
    """HTTP client for the sensor service with resilience patterns.

    Uses a circuit breaker (pybreaker) to prevent cascade failures and
    tenacity for retry with exponential backoff. Falls back gracefully
    when the sensor service is unavailable.
    """

    def __init__(
        self,
        base_url: str,
        api_token: str,
        cb_fail_max: int = 5,
        cb_reset_timeout: int = 30,
    ):
        self.base_url = base_url
        self.api_token = api_token
        self.breaker = pybreaker.CircuitBreaker(
            fail_max=cb_fail_max,
            reset_timeout=cb_reset_timeout,
            name="sensor-service",
        )

    @retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=1, max=4))
    def _make_request(self, sensor_id: str) -> dict | None:
        """Make HTTP request to sensor service with retry logic.

        Retries up to 3 times with exponential backoff:
        multiplier * 2^(attempt-1), so 1s, 2s, 4s with multiplier=1.
        Timeout per request is 2 seconds.
        """
        response = httpx.get(
            f"{self.base_url}/sensors/{sensor_id}",
            headers={"Authorization": f"Bearer {self.api_token}"},
            timeout=2.0,
        )
        if response.status_code == 404:
            return None
        response.raise_for_status()
        return response.json()

    def get_sensor(self, sensor_id: str) -> tuple[dict | None, bool]:
        """Get sensor data via circuit breaker.

        Returns:
            A tuple of (sensor_data, is_validated) with three possible outcomes:
            - (sensor_data, True): Sensor found and validated successfully
            - (None, True): Sensor confirmed not found (404 from sensor service)
            - (None, False): Sensor service unavailable — fallback, not validated
        """
        try:
            result = self.breaker.call(self._make_request, sensor_id)
            return result, True
        except pybreaker.CircuitBreakerError:
            logger.warning(
                "Circuit breaker is OPEN — sensor service unavailable",
                extra={"sensor_id": sensor_id, "breaker_state": "open"},
            )
            return None, False
        except Exception as e:
            logger.warning(
                "Sensor service request failed: %s",
                str(e),
                extra={"sensor_id": sensor_id},
            )
            return None, False
