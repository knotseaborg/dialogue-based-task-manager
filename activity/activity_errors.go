package activity

type NoActivityError struct{}

func (e NoActivityError) Error() string {
	return "no activity found in the database"
}

type FatalError struct{}

func (e FatalError) Error() string {
	return "Fatal error. Please attempt self-repair"
}
