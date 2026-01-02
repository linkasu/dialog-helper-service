package config

import "fmt"

func errMissing(key string) error {
	return fmt.Errorf("missing required env %s", key)
}
