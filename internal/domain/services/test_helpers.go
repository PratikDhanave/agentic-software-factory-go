package services

type MockLogger struct{}

func (ml *MockLogger) Info(msg string, fields ...interface{}) {}
func (ml *MockLogger) Error(msg string, err error, fields ...interface{}) {}
func (ml *MockLogger) Debug(msg string, fields ...interface{}) {}
func (ml *MockLogger) Warn(msg string, fields ...interface{}) {}
