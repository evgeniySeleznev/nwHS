package app

// Config определяет конфигурацию customer-service.
type Config struct {
	ServiceName string `mapstructure:"service_name"`

	GRPC struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"grpc"`

	Postgres struct {
		DSN            string `mapstructure:"dsn"`
		MaxConns       int32  `mapstructure:"max_conns"`
		MigrationsPath string `mapstructure:"migrations_path"`
	} `mapstructure:"postgres"`

	Kafka struct {
		Brokers            []string `mapstructure:"brokers"`
		CustomerTopic      string   `mapstructure:"customer_topic"`
		OutboxPollInterval string   `mapstructure:"outbox_poll_interval"`
		ConsumerGroup      string   `mapstructure:"consumer_group"`
	} `mapstructure:"kafka"`

	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
		TTL      string `mapstructure:"ttl"`
	} `mapstructure:"redis"`

	Search struct {
		Endpoint string `mapstructure:"endpoint"`
		Index    string `mapstructure:"index"`
	} `mapstructure:"search"`
}

// Defaults заполняет значения по умолчанию.
func (c *Config) Defaults() {
	if c.ServiceName == "" {
		c.ServiceName = "customer-service"
	}
	if c.GRPC.Port == 0 {
		c.GRPC.Port = 50051
	}
	if c.Redis.TTL == "" {
		c.Redis.TTL = "10m"
	}
	if c.Kafka.OutboxPollInterval == "" {
		c.Kafka.OutboxPollInterval = "500ms"
	}
	if c.Search.Index == "" {
		c.Search.Index = "customers"
	}
	if c.Postgres.MaxConns == 0 {
		c.Postgres.MaxConns = 16
	}
}
