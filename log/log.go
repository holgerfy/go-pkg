package log

import (
	"context"
	"fmt"
	"github.com/holgerfy/go-pkg/funcs"
	"github.com/holgerfy/go-pkg/unique"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"strconv"
	"time"
)

type Log struct {
	logger *zap.Logger
}

const loggerKey = iota

var log = &Log{}

func Start() {
	level := zap.DebugLevel
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(getWriter())),
		level,
	)
	log.logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}

func getWriter() io.Writer {
	logDir := funcs.GetRoot()
	fmt.Println(logDir + "/%Y%m%d.log")
	logWriter, _ := rotatelogs.New(logDir+"/%Y%m%d.log",
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour*24),
	)
	return logWriter
}

func Logger() *Log {
	return log
}

func WithFields(ctx context.Context, fields map[string]string) context.Context {
	fieldArr := make([]zap.Field, 0)
	for k, v := range fields {
		f := zap.String(k, v)
		fieldArr = append(fieldArr, f)
	}
	l := WithCtx(ctx)
	fmt.Println(l)
	return context.WithValue(ctx, loggerKey, l.With(fieldArr...))
}

func NewContext(ctx context.Context, fields ...zapcore.Field) context.Context {
	return context.WithValue(ctx, loggerKey, WithCtx(ctx).With(fields...))
}

func WithCtx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return log.logger
	}
	fmt.Println("csdfsf----", ctx)
	if ctxLogger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		fmt.Println("sdfs-has:  ", ctxLogger)
		return ctxLogger
	}
	return log.logger
}

func (l *Log) Info(ctx context.Context, args ...interface{}) {
	WithCtx(ctx).Info(fmt.Sprint(args))
}

func (l *Log) Error(ctx context.Context, args ...interface{}) {
	WithCtx(ctx).Error(fmt.Sprint(args))
}

func (l *Log) Debug(ctx context.Context, args ...interface{}) {
	WithCtx(ctx).Debug(fmt.Sprint(args))
}

func (l *Log) Warn(ctx context.Context, args ...interface{}) {
	WithCtx(ctx).Error(fmt.Sprint(args))
}

func (l *Log) Fatal(ctx context.Context, args ...interface{}) {
	WithCtx(ctx).Fatal(fmt.Sprint(args))
}

func GrpcUnaryServerInterceptor(l *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		startTime := time.Now()
		reqId := ""
		headers, ok := metadata.FromIncomingContext(ctx)
		if ok {
			reqIdArr := headers.Get("Req-Id")
			if len(reqIdArr) > 0 {
				reqId = reqIdArr[0]
			}
		}
		if reqId == "" {
			reqId = unique.Uuid()
		}
		items := map[string]string{
			"method": info.FullMethod,
			"req-id": reqId,
		}
		newCtx := WithFields(ctx, items)
		resp, err = handler(newCtx, req)
		code := status.Code(err)
		duration := time.Since(startTime)

		items1 := map[string]string{
			"code":     strconv.Itoa(int(code)),
			"duration": duration.String(),
		}
		fields := make([]zap.Field, 0)
		for k, v := range items1 {
			f := zap.String(k, v)
			fields = append(fields, f)
		}
		WithCtx(newCtx).Info("serverLogger", fields...)
		return resp, err
	}
}
