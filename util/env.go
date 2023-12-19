package util

import (
	"github.com/nkhoaa96/go-base/storage/local"
	"strings"
)

// IsDevEnv check if a container is running under non-prod environment.
func IsDevEnv() bool {
	var env = local.Getenv("ENVIRONMENT")
	return strings.EqualFold(env, "dev") ||
		strings.EqualFold(env, "fat") ||
		strings.EqualFold(env, "uat")
}

func IsLocal() bool {
	var env = local.Getenv("ENVIRONMENT")
	return strings.EqualFold(env, "local")
}
