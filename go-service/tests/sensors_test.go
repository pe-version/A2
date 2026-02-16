package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"iot-sensor-service/database"
	"iot-sensor-service/handlers"
	"iot-sensor-service/middleware"
	"iot-sensor-service/repositories"
)

const testToken = "test-secret-token"

// setupTestRouter creates a test router with a temporary database.
func setupTestRouter(t *testing.T) (*gin.Engine, func()) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	dbPath := tmpFile.Name()

	// Connect to database
	db, err := database.Connect(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize schema
	if err := database.InitSchema(db); err != nil {
		db.Close()
		os.Remove(dbPath)
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create repository and handlers
	sensorRepo := repositories.NewSQLiteSensorRepository(db)
	healthHandler := handlers.NewHealthHandler()
	sensorHandler := handlers.NewSensorHandler(sensorRepo)

	// Set up router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health endpoint - no auth required (for load balancer probes)
	router.GET("/health", healthHandler.Health)

	// Protected routes - require Bearer token
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(testToken))
	protected.GET("/sensors", sensorHandler.ListSensors)
	protected.GET("/sensors/:id", sensorHandler.GetSensor)
	protected.POST("/sensors", sensorHandler.CreateSensor)
	protected.PUT("/sensors/:id", sensorHandler.UpdateSensor)
	protected.DELETE("/sensors/:id", sensorHandler.DeleteSensor)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return router, cleanup
}

func TestUnauthorizedWithoutToken(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestUnauthorizedWithInvalidToken(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAuthorizedWithValidToken(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHealthEndpointNoAuthRequired(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	// No Authorization header - health should work without auth
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
	if response["service"] != "go" {
		t.Errorf("Expected service 'go', got '%s'", response["service"])
	}
}

func TestListSensorsEmpty(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	sensors := response["sensors"].([]interface{})
	if len(sensors) != 0 {
		t.Errorf("Expected empty sensors list, got %d", len(sensors))
	}

	count := int(response["count"].(float64))
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestCreateSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	newSensor := map[string]interface{}{
		"name":     "Test Sensor",
		"type":     "temperature",
		"location": "test_room",
		"value":    72.5,
		"unit":     "fahrenheit",
		"status":   "active",
	}
	body, _ := json.Marshal(newSensor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sensors", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)

	if created["id"] == nil {
		t.Error("Expected id in response")
	}
	if created["name"] != "Test Sensor" {
		t.Errorf("Expected name 'Test Sensor', got '%s'", created["name"])
	}
}

func TestCreateAndFetchSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create
	newSensor := map[string]interface{}{
		"name":     "Fetch Test Sensor",
		"type":     "humidity",
		"location": "bathroom",
		"value":    65.0,
		"unit":     "percent",
		"status":   "active",
	}
	body, _ := json.Marshal(newSensor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sensors", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d", w.Code)
	}

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	sensorID := created["id"].(string)

	// Fetch
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/sensors/"+sensorID, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var fetched map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &fetched)

	if fetched["id"] != sensorID {
		t.Errorf("Expected id '%s', got '%s'", sensorID, fetched["id"])
	}
	if fetched["name"] != "Fetch Test Sensor" {
		t.Errorf("Expected name 'Fetch Test Sensor', got '%s'", fetched["name"])
	}
}

func TestUpdateSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create
	newSensor := map[string]interface{}{
		"name":     "Update Test",
		"type":     "temperature",
		"location": "kitchen",
		"value":    70.0,
		"unit":     "fahrenheit",
		"status":   "active",
	}
	body, _ := json.Marshal(newSensor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sensors", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	sensorID := created["id"].(string)

	// Update
	updateData := map[string]interface{}{
		"value":  75.5,
		"status": "inactive",
	}
	body, _ = json.Marshal(updateData)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/sensors/"+sensorID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &updated)

	if updated["value"].(float64) != 75.5 {
		t.Errorf("Expected value 75.5, got %v", updated["value"])
	}
	if updated["status"] != "inactive" {
		t.Errorf("Expected status 'inactive', got '%s'", updated["status"])
	}
	if updated["name"] != "Update Test" {
		t.Errorf("Expected name 'Update Test' (unchanged), got '%s'", updated["name"])
	}
}

func TestDeleteSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create
	newSensor := map[string]interface{}{
		"name":     "Delete Test",
		"type":     "motion",
		"location": "hallway",
		"value":    0.0,
		"unit":     "boolean",
		"status":   "active",
	}
	body, _ := json.Marshal(newSensor)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sensors", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d - %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	sensorID := created["id"].(string)

	// Delete
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/sensors/"+sensorID, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}

	// Verify deleted
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/sensors/"+sensorID, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestGetNonexistentSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors/nonexistent-id", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestDeleteNonexistentSensor(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/sensors/nonexistent-id", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestListSensorsAfterCreate(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create multiple sensors
	for i := 0; i < 3; i++ {
		sensor := map[string]interface{}{
			"name":     "Sensor",
			"type":     "temperature",
			"location": "room",
			"value":    70.0 + float64(i),
			"unit":     "fahrenheit",
			"status":   "active",
		}
		body, _ := json.Marshal(sensor)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/sensors", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}

	// List
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/sensors", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	count := int(response["count"].(float64))
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	sensors := response["sensors"].([]interface{})
	if len(sensors) != 3 {
		t.Errorf("Expected 3 sensors, got %d", len(sensors))
	}
}
