"""Integration tests for sensor CRUD operations."""

import pytest


class TestAuthentication:
    """Tests for authentication middleware."""

    def test_unauthorized_without_token(self, client):
        """Requests without token should return 401."""
        response = client.get("/sensors")
        assert response.status_code in (401, 403)

    def test_unauthorized_with_invalid_token(self, client):
        """Requests with invalid token should return 401."""
        response = client.get(
            "/sensors",
            headers={"Authorization": "Bearer invalid-token"},
        )
        assert response.status_code == 401

    def test_authorized_with_valid_token(self, client, auth_headers):
        """Requests with valid token should succeed."""
        response = client.get("/sensors", headers=auth_headers)
        assert response.status_code == 200


class TestHealthEndpoint:
    """Tests for the health endpoint."""

    def test_health_returns_ok_without_auth(self, client):
        """Health endpoint should return ok status without authentication."""
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "ok"
        assert data["service"] == "python"


class TestSensorCRUD:
    """Tests for sensor CRUD operations."""

    def test_list_sensors_empty(self, client, auth_headers):
        """Empty database should return empty list."""
        response = client.get("/sensors", headers=auth_headers)
        assert response.status_code == 200
        data = response.json()
        assert data["sensors"] == []
        assert data["count"] == 0

    def test_create_sensor(self, client, auth_headers):
        """Creating a sensor should return 201 with the created sensor."""
        new_sensor = {
            "name": "Test Sensor",
            "type": "temperature",
            "location": "test_room",
            "value": 72.5,
            "unit": "fahrenheit",
            "status": "active",
        }
        response = client.post("/sensors", json=new_sensor, headers=auth_headers)
        assert response.status_code == 201
        data = response.json()
        assert "id" in data
        assert data["name"] == "Test Sensor"
        assert data["type"] == "temperature"
        assert data["value"] == 72.5

    def test_create_and_fetch_sensor(self, client, auth_headers):
        """Created sensor should be retrievable by ID."""
        # Create
        new_sensor = {
            "name": "Fetch Test Sensor",
            "type": "humidity",
            "location": "bathroom",
            "value": 65.0,
            "unit": "percent",
            "status": "active",
        }
        create_response = client.post("/sensors", json=new_sensor, headers=auth_headers)
        assert create_response.status_code == 201
        created = create_response.json()
        sensor_id = created["id"]

        # Fetch
        get_response = client.get(f"/sensors/{sensor_id}", headers=auth_headers)
        assert get_response.status_code == 200
        fetched = get_response.json()
        assert fetched["id"] == sensor_id
        assert fetched["name"] == "Fetch Test Sensor"
        assert fetched["value"] == 65.0

    def test_update_sensor(self, client, auth_headers):
        """Updating a sensor should modify only specified fields."""
        # Create
        new_sensor = {
            "name": "Update Test",
            "type": "temperature",
            "location": "kitchen",
            "value": 70.0,
            "unit": "fahrenheit",
            "status": "active",
        }
        create_response = client.post("/sensors", json=new_sensor, headers=auth_headers)
        sensor_id = create_response.json()["id"]

        # Update
        update_data = {"value": 75.5, "status": "inactive"}
        update_response = client.put(
            f"/sensors/{sensor_id}", json=update_data, headers=auth_headers
        )
        assert update_response.status_code == 200
        updated = update_response.json()
        assert updated["value"] == 75.5
        assert updated["status"] == "inactive"
        assert updated["name"] == "Update Test"  # Unchanged

    def test_delete_sensor(self, client, auth_headers):
        """Deleting a sensor should remove it from the database."""
        # Create
        new_sensor = {
            "name": "Delete Test",
            "type": "motion",
            "location": "hallway",
            "value": 0,
            "unit": "boolean",
            "status": "active",
        }
        create_response = client.post("/sensors", json=new_sensor, headers=auth_headers)
        sensor_id = create_response.json()["id"]

        # Delete
        delete_response = client.delete(f"/sensors/{sensor_id}", headers=auth_headers)
        assert delete_response.status_code == 204

        # Verify deleted
        get_response = client.get(f"/sensors/{sensor_id}", headers=auth_headers)
        assert get_response.status_code == 404

    def test_get_nonexistent_sensor(self, client, auth_headers):
        """Getting a nonexistent sensor should return 404."""
        response = client.get("/sensors/nonexistent-id", headers=auth_headers)
        assert response.status_code == 404

    def test_delete_nonexistent_sensor(self, client, auth_headers):
        """Deleting a nonexistent sensor should return 404."""
        response = client.delete("/sensors/nonexistent-id", headers=auth_headers)
        assert response.status_code == 404

    def test_create_sensor_validation(self, client, auth_headers):
        """Creating a sensor with invalid data should return 422."""
        invalid_sensor = {
            "name": "Invalid",
            "type": "invalid_type",  # Not a valid type
            "location": "test",
            "value": 0,
            "unit": "test",
            "status": "active",
        }
        response = client.post("/sensors", json=invalid_sensor, headers=auth_headers)
        assert response.status_code == 422

    def test_list_sensors_after_create(self, client, auth_headers):
        """List should include created sensors."""
        # Create multiple sensors
        for i in range(3):
            sensor = {
                "name": f"Sensor {i}",
                "type": "temperature",
                "location": f"room_{i}",
                "value": 70.0 + i,
                "unit": "fahrenheit",
                "status": "active",
            }
            client.post("/sensors", json=sensor, headers=auth_headers)

        # List
        response = client.get("/sensors", headers=auth_headers)
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 3
        assert len(data["sensors"]) == 3
