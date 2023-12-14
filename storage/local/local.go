package local

import (
	"os"
	"syscall"

	"github.com/joho/godotenv"
)

func LoadEnv(fileNames ...string) error {
	return godotenv.Load(fileNames...)
}

func Getenv(key string) string {
	return os.Getenv(key)
}

func MustMapEnv(target *string, envKey string) {
	v := Getenv(envKey)
	if v == "" {
		return
	}
	*target = v
}
func Setenv(key, value string) error {
	err := syscall.Setenv(key, value)
	if err != nil {
		return err
	}
	return nil
}
