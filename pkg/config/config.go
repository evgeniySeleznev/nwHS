package config

import (
    "fmt"
    "strings"

    "github.com/spf13/viper"
)

// Loader отвечает за загрузку конфигурации из env-файлов и переменных окружения.
type Loader struct {
    prefix string
    paths  []string
}

// Option задаёт параметры загрузчика.
type Option func(*Loader)

// WithPrefix задаёт префикс переменных окружения.
func WithPrefix(prefix string) Option {
    return func(l *Loader) {
        l.prefix = prefix
    }
}

// WithConfigPaths добавляет пути к конфигурационным файлам.
func WithConfigPaths(paths ...string) Option {
    return func(l *Loader) {
        l.paths = append(l.paths, paths...)
    }
}

// WithDecoder позволяет переопределить env decoder.
// New создаёт загрузчик конфигурации.
func New(opts ...Option) *Loader {
    l := &Loader{}

    for _, opt := range opts {
        opt(l)
    }

    return l
}

// Load читает конфигурацию в структуру dst.
func (l *Loader) Load(dst interface{}) error {
    if dst == nil {
        return fmt.Errorf("config: dst is nil")
    }

    v := viper.New()
    v.SetEnvPrefix(l.prefix)
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    if len(l.paths) == 0 {
        v.SetConfigName("config")
        v.SetConfigType("yaml")
        v.AddConfigPath(".")
    } else {
        for _, path := range l.paths {
            v.AddConfigPath(path)
        }
        v.SetConfigName("config")
    }

    if err := v.ReadInConfig(); err != nil {
        // допускаем отсутствие файла, rely на env
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("config: read file: %w", err)
        }
    }

    for _, key := range v.AllKeys() {
        if err := v.BindEnv(key); err != nil {
            return fmt.Errorf("config: bind env %s: %w", key, err)
        }
    }

    if err := v.Unmarshal(dst); err != nil {
        return fmt.Errorf("config: unmarshal file: %w", err)
    }

    return nil
}

