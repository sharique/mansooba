package handler_test

import (
	"os"
	"testing"

	"github.com/sharique/mansooba/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init("debug")
	os.Exit(m.Run())
}
