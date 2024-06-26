package main

import (
	"os"
	"strconv"
	"strings"
)

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
//
//lint:ignore U1000 We will be using this function in the future
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

// Helper to read an environment variable into a bool or return default value
//
//lint:ignore U1000 We will be using this function in the future
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}

// Helper to read an environment variable into a string slice or return default value
//
//lint:ignore U1000 We will be using this function in the future
func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valStr := strings.TrimSpace(getEnv(name, ""))
	if valStr == "" {
		return defaultVal
	}

	val := strings.Split(valStr, sep)

	return val
}
