package redis

//
//import (
//	"github.com/nkhoaa96/go-base/cache"
//	"github.com/nkhoaa96/go-base/storage/local"
//	"strconv"
//)
//
//func GetRedisConnection(aiClient appinsights.Client) (cache.Cacher, error) {
//	password, err := keyvault.GetSecretValue(keyvault.RedisPassword)
//	if err != nil {
//		return nil, err
//	}
//	enableRedisSSL, _ := strconv.ParseBool(local.Getenv("ENABLE_REDIS_SSL"))
//	return cache.New(local.Getenv(RedisAddress), *password, cache.SetAIClient(aiClient), cache.EnableSSL(enableRedisSSL))
//}
