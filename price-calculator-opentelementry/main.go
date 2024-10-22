package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
)

// Global variables for base price and tax rate
var basePrice float64
var taxRate float64
var tracer trace.Tracer

// PriceRequest structure for input data
type PriceRequest struct {
	BasePrice float64 `json:"base_price"`
	TaxRate   float64 `json:"tax_rate"`
}

// PriceResponse structure for output data
type PriceResponse struct {
	TotalPrice float64 `json:"total_price"`
}

func main() {
	fmt.Println("Price Calculator Project")
	ctx := context.Background()

	// Initialize OpenTelemetry
	cleanup, err := initOpenTelemetry(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	defer cleanup() // Ensure resources are cleaned up on exit

	// Initialize Gorilla Mux router
	router := mux.NewRouter()

	// Define the API endpoints with OpenTelemetry tracing
	router.Handle("/calculate", otelhttp.NewHandler(http.HandlerFunc(calculatePrice), "CalculatePrice")).Methods("POST")
	router.Handle("/setBasePrice/{value}", otelhttp.NewHandler(http.HandlerFunc(setBasePrice), "SetBasePrice")).Methods("POST")
	router.Handle("/setTaxRate/{value}", otelhttp.NewHandler(http.HandlerFunc(setTaxRate), "SetTaxRate")).Methods("POST")

	// Start the HTTP server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// Initializes OpenTelemetry
func initOpenTelemetry(ctx context.Context) (func(), error) {
	// Create the OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint("localhost:4318"), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %v", err)
	}

	// Define the resource attributes (e.g., service name)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("price-calculator"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	// Create the trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter), // OTLP exporter
		sdktrace.WithResource(res),
	)

	// Set the global tracer provider
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("price-calculator") // Create a tracer for the application

	// Return a cleanup function to shutdown the tracer provider
	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

// Calculates the total price based on the base price and tax rate
func calculatePrice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start a new span for the price calculation
	_, span := tracer.Start(ctx, "CalculateTotalPrice")
	defer span.End() // Ensure the span is ended when the function exits

	// Create a PriceRequest struct with current values
	request := PriceRequest{BasePrice: basePrice, TaxRate: taxRate}

	// Simulate processing delay for tracing visibility
	time.Sleep(100 * time.Millisecond)

	// Calculate the total price
	totalPrice := request.BasePrice + (request.BasePrice * request.TaxRate / 100)

	// Prepare the response
	response := PriceResponse{TotalPrice: totalPrice}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Printf("Calculated total price: %f", totalPrice)
}

// Sets the base price from the request
func setBasePrice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	value := vars["value"]

	var err error
	basePrice, err = strconv.ParseFloat(value, 64) // Parse the base price from the URL
	if err != nil {
		log.Printf("Invalid base price: %v", err)
		http.Error(w, "Invalid base price", http.StatusBadRequest)
		return
	}

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Base price set"}); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Printf("Base price set to: %f", basePrice)
}

// Sets the tax rate from the request
func setTaxRate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	value := vars["value"]

	var err error
	taxRate, err = strconv.ParseFloat(value, 64) // Parse the tax rate from the URL
	if err != nil {
		log.Printf("Invalid tax rate: %v", err)
		http.Error(w, "Invalid tax rate", http.StatusBadRequest)
		return
	}

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Tax rate set"}); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Printf("Tax rate set to: %f", taxRate)
}
