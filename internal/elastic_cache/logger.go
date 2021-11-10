package elastic_cache

import (
	"fmt"
	"go.uber.org/zap"
)

type ElasticLogger struct {
}

func (l ElasticLogger) Printf(format string, v ...interface{}) {
	zap.L().Error(fmt.Sprintf(format, v))
}
