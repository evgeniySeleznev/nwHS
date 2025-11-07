package sentry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

// Config описывает настройки подключения к Sentry.
type Config struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
}

// Client управляет жизненным циклом Sentry SDK.
type Client struct {
	enabled bool
}

// Init настраивает глобальный Sentry SDK и возвращает управляемый клиент.
func Init(cfg Config) (*Client, error) {
	if cfg.DSN == "" {
		return &Client{enabled: false}, nil
	}

	opts := sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
		EnableTracing:    true,
	}

	if opts.SampleRate == 0 {
		opts.SampleRate = 1.0
	}
	if opts.TracesSampleRate == 0 {
		opts.TracesSampleRate = 1.0
	}

	if err := sentry.Init(opts); err != nil {
		return nil, err
	}

	return &Client{enabled: true}, nil
}

// CaptureError отправляет исключение в Sentry.
func (c *Client) CaptureError(err error) {
	if !c.enabled || err == nil {
		return
	}
	sentry.CaptureException(err)
}

// CaptureMessage отправляет произвольное сообщение в Sentry.
func (c *Client) CaptureMessage(msg string) {
	if !c.enabled || msg == "" {
		return
	}
	sentry.CaptureMessage(msg)
}

// Flush ожидает отправку событий в Sentry.
func (c *Client) Flush(ctx context.Context) {
	if !c.enabled {
		return
	}

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	sentry.Flush(timeout)
}

// Enabled сообщает, активирован ли Sentry.
func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}
