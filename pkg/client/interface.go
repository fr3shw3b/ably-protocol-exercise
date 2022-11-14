package client

type Client interface {
	Connect() error
	Close() error
	Result() Result
}
