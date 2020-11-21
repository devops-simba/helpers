package helpers

import "os"

// ReadEnv Read an environment variable or a default value
func ReadEnv(envName, defaultValue string) string {
	value, ok := os.LookupEnv(envName)
	if !ok {
		value = defaultValue
	}
	return value
}
