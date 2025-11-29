package daygo

type Logger interface {
	Debug(any, ...any)
	Info(any, ...any)
	Warn(any, ...any)
	Error(any, ...any)
	Fatal(any, ...any)
}

type NoOpLogger struct{}

func (n NoOpLogger) Debug(_ any, _ ...any) {}

func (n NoOpLogger) Info(_ any, _ ...any) {}

func (n NoOpLogger) Warn(_ any, _ ...any) {}

func (n NoOpLogger) Error(_ any, _ ...any) {}

func (n NoOpLogger) Fatal(_ any, _ ...any) {}
