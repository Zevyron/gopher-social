package env

import (
	"os"
	"strconv"
)

func GetString(key, fallback string) string {
	val, err := os.LookupEnv(key)

	if !err {
		return fallback
	}

	return val
}

func GetInt(key string, fallback int) int {
	val, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}

	return valAsInt
}