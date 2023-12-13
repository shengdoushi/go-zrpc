package zrpc

import (
	"context"
	"fmt"
	"github.com/pebbe/zmq4"
	"testing"
)

func startZrpcServer(endpoint string) *socket {
	s, _ := newSocket(zmq4.ROUTER)

	s.newChannel = func(ch *channel) {
		fmt.Println("router server callback: ")
		go func() {
			ev, err := ch.RecvEvent()
			if err != nil {
				fmt.Println("router server recv event error: ", err.Error())
				return
			}

			fmt.Println("router server recv: ", ev)
			ev2 := new(event)
			ev2.Name = "GetUserById"
			ev2.Args = "mynamezrpc"
			ch.SendEvent(ev2)

			s.CloseChannel(ch.id)
		}()
	}
	s.Bind(endpoint)
	return s
}

func startZrpcClientDemo(endpoint string) {
	c, _ := newSocket(zmq4.DEALER)
	defer c.Close()

	c.Connect(endpoint)

	ev := new(event)
	ev.Hdr.MsgID = randomString(32)
	ev.Hdr.Version = ProtocolVersion
	ev.Name = "GetUserById"
	ev.Args = 12
	fmt.Println("dealer client send msg: ", ev)
	ch := c.openChannel(ev.Hdr.MsgID, "")
	err := ch.SendEvent(ev)
	if err != nil {
		fmt.Println("dealer client send event error: ", err.Error())
		return
	}
	fmt.Println("dealer client send event ok")

	rep, err := ch.RecvEvent()
	if err != nil {
		fmt.Println("dealer client recv event error: ", err.Error())
		return
	}
	c.CloseChannel(ev.Hdr.MsgID)

	fmt.Println("dealder client recv event: ", rep)
}

func TestZrpcDemo(t *testing.T) {
	closeCh := make(chan struct{})
	defer close(closeCh)

	endpoint := "inproc://tetsclient"
	go func() {
		s := startZrpcServer(endpoint)
		defer s.Close()
		<-closeCh
	}()

	startZrpcClientDemo(endpoint)
}

func startMyServer(endpoint string) *Server {
	server := NewServer(zmq4.ROUTER)

	RegisterDemoServer(server, NewMyDemoServer())
	server.Bind(endpoint)
	return server
}

func startMyClientDemo(endpoint string) {
	conn := NewClient(zmq4.DEALER)
	defer conn.Close()

	conn.Connect(endpoint)
	c := NewMyDemoClient(conn)

	rep, err := c.GetUserById(12)
	if err != nil {
		fmt.Println("GetUserById error: ", err)
		return
	}

	fmt.Println("GetUserById result: ", rep)
}

func TestMyDemo(t *testing.T) {
	closeCh := make(chan struct{})
	defer close(closeCh)

	endpoint := "inproc://tetsclient"
	go func() {
		server := startMyServer(endpoint)
		defer server.Close()
		<-closeCh
	}()
	startMyClientDemo(endpoint)
}

type Client struct {
	sock *socket
}

func (c *Client) Invoke(ctx context.Context, method string, args any) (any, error) {
	ev := new(event)
	ev.Hdr.MsgID = randomString(32)
	ev.Hdr.Version = ProtocolVersion
	ev.Name = method
	ev.Args = args
	fmt.Println("dealer client send msg: ", ev)
	ch := c.sock.openChannel(ev.Hdr.MsgID, "")
	defer c.sock.CloseChannel(ev.Hdr.MsgID)

	err := ch.SendEvent(ev)
	if err != nil {
		fmt.Println("dealer client send event error: ", err.Error())
		return nil, err
	}
	fmt.Println("dealer client send event ok")

	rep, err := ch.RecvEvent()
	if err != nil {
		fmt.Println("dealer client recv event error: ", err.Error())
		return nil, err
	}

	return rep.Args, nil
}

func (c *Client) Close() error {
	return c.sock.Close()
}
func (c *Client) Connect(endpoint string) error {
	return c.sock.Connect(endpoint)
}

func NewClient(zmqType zmq4.Type) *Client {
	c := new(Client)
	sock, _ := newSocket(zmqType)
	c.sock = sock
	return c
}

type MyDemoClient struct {
	client *Client
}

func (s *MyDemoClient) GetUserById(id int) (string, error) {
	rep, err := s.client.Invoke(context.Background(), "GetUserById", id)
	if err != nil {
		return "", err
	}
	return rep.(string), nil
}

func NewMyDemoClient(client *Client) *MyDemoClient {
	return &MyDemoClient{
		client: client,
	}
}

type Server struct {
	sock            *socket
	methodCallbacks map[string]func(event2 *event) (*event, error)
}

func (s *Server) Bind(endpoint string) error {
	return s.sock.Bind(endpoint)
}

func (s *Server) RegisterMethod(method string, fn func(event2 *event) (*event, error)) {
	s.methodCallbacks[method] = fn
}
func (s *Server) Close() error {
	return s.sock.Close()
}

func NewServer(zmqType zmq4.Type) *Server {
	sock, _ := newSocket(zmqType)
	s := &Server{sock: sock}
	s.methodCallbacks = make(map[string]func(event2 *event) (*event, error))

	sock.newChannel = func(ch *channel) {
		go func() {
			fmt.Println("router server opened new channel", ch.id)
			ev, err := ch.RecvEvent()
			defer sock.CloseChannel(ch.id)

			if err != nil {
				fmt.Println("router server recv event error: ", err.Error())
				return
			}

			fn := s.methodCallbacks[ev.Name]
			if fn != nil {
				evRep, err := fn(ev)
				if err == nil && evRep != nil {
					ch.SendEvent(evRep)
				}
			}
		}()
	}
	return s
}

type MyDemoServerInterface interface {
	GetUserById(id int64) (string, error)
}

func RegisterDemoServer(s *Server, serverInterface MyDemoServerInterface) {
	s.RegisterMethod("GetUserById", func(event2 *event) (*event, error) {
		rep, err := serverInterface.GetUserById(event2.Args.(int64))
		if err != nil {
			return nil, err
		}
		evRep := new(event)
		evRep.Name = event2.Name
		evRep.Args = rep
		return evRep, nil
	})
}

func NewMyDemoServer() *MyDemoServer {
	return &MyDemoServer{}
}

type MyDemoServer struct {
}

func (s *MyDemoServer) GetUserById(id int64) (string, error) {
	fmt.Println("MyDemoServer GetUserById(", id, "), return ", "mynamedemo")
	return "mynamedemo", nil
}
