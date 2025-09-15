package s3driver

type mockNotFoundError struct{}

func (m *mockNotFoundError) Error() string     { return "NotFound" }
func (m *mockNotFoundError) ErrorCode() string { return "NotFound" }
