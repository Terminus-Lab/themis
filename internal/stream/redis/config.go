package redis

type RedisStreamConfig struct {
	RedisAddr     string
	RedisPassword string
	Stream        string
	Group         string
	ConsumerName  string
}

func NewRedisStreamConfig(redisAddr string, redisPassword string, stream string, group string, consumerName string) *RedisStreamConfig {
	return &RedisStreamConfig{
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		Stream:        stream,
		Group:         group,
		ConsumerName:  consumerName,
	}
}
