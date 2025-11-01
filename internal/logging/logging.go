package logging

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type Component string

const (
	MainComponent            Component = "MAIN"
	ApiComponent             Component = "API"
	PredictorClientComponent Component = "PREDICTOR_CLIENT"
	PredictorComponent       Component = "PREDICTOR"
)

type Logger struct {
	*logrus.Entry
}

func NewLogger(cfg config.Config) *Logger {
	log := logrus.New()

	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	switch strings.ToLower(cfg.Log.Format) {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	if cfg.Log.File != "" {
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, file))
		} else {
			log.SetOutput(os.Stdout)
		}
	} else {
		log.SetOutput(os.Stdout)
	}

	return &Logger{
		Entry: logrus.NewEntry(log).WithField("component", MainComponent),
	}
}

func (l *Logger) WithComponent(component Component) *Logger {
	return &Logger{
		Entry: l.Entry.WithField("component", component),
	}
}

func (l *Logger) WithApiTag() *Logger {
	return l.WithComponent(ApiComponent)
}

func (l *Logger) WithPredictorClientTag() *Logger {
	return l.WithComponent(PredictorClientComponent)
}

func (l *Logger) WithPredictorTag() *Logger {
	return l.WithComponent(PredictorComponent)
}

func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{
		Entry: l.Entry.WithField(key, value),
	}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := logrus.Fields{}

	for key := range utils.ContextKeys {
		if val, ok := utils.GetContextValue(ctx, key); ok && val != nil {
			switch key {
			case utils.UserCtxKey:
				if user, ok := val.(*models.User); ok {
					fields["user_id"] = user.ID.String()
					fields["user_login"] = user.Login
				}
			case utils.RequestIDKey:
				if reqID, ok := val.(string); ok && reqID != "" {
					fields["request_id"] = reqID
				}
			case utils.TimeKey:
				continue
			case utils.PathKey:
				if path, ok := val.(string); ok && path != "" {
					fields["path"] = path
				}
			case utils.MethodKey:
				if method, ok := val.(string); ok && method != "" {
					fields["method"] = method
				}
			case utils.RequestBodyKey:
				continue
			}
		}
	}

	if len(fields) > 0 {
		return &Logger{
			Entry: l.WithFields(fields),
		}
	}

	return l
}
