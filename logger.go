package daygo

type Logger interface {
	Debug(interface{}, ...interface{})
	Info(interface{}, ...interface{})
	Warn(interface{}, ...interface{})
	Error(interface{}, ...interface{})
	Fatal(interface{}, ...interface{})
}
