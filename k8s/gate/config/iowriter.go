package config

// type Writer interface {
// 	Write(p []byte) (n int, err error)
// }

// type LogConverter struct {
// 	log *slog.Logger
// }

// func (l *LogConverter) Write(p []byte) (n int, err error) {
// 	l.log.LogAttrs(context.Background(), slog.LevelDebug,
// 		string(p),
// 		logging.LogAttrCategory(logging.LogCategoryK8s),
// 	)
// 	return len(p), nil
// }

// func NewIOWriter(logger *slog.Logger) io.Writer {
// 	return &LogConverter{log: logger}
// }
