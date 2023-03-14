package notification

type SmsSender interface {
	Send(string, string) error
}
