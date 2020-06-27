package synccloser_test

import (
	"fmt"

	"github.com/Darigaaz/synccloser"
)

type Service struct {
	// embed SyncCloser
	synccloser.SyncCloser

	C chan int
}

func NewService() *Service {
	return &Service{
		// instantiate SyncCloser
		// currently its not possible to use zero value
		// because we need to make channels in race free manner
		SyncCloser: *synccloser.New(),

		C: make(chan int),
	}
}

func (service *Service) DoStuff(n int) {
	service.C <- n
}

func (service *Service) Start() {
	var lastError error
	closeChannel := service.CloseChannel()

	for {
		select {
		// listen to closeChannel in main loop
		case done := <-closeChannel:
			fmt.Println("received close command, cleaning up")
			lastError = service.cleanup()

			// response with error when you are done cleaning up
			fmt.Println("sending Done")
			done(lastError)

			fmt.Println("really done, service closed")

			// dont forget to return
			return

		//	other cases
		case v:= <-service.C:
			fmt.Println(v)
		}
	}
}

func (service *Service) cleanup() error {
	// cleanup logic goes here

	return nil
}

// You can omit this method
func (service *Service) Close() error {
	// custom logic goes here

	return service.SyncCloser.Close()
}

func ExampleSyncCloser() {
	service := NewService()

	go service.Start()

	service.DoStuff(1)
	service.DoStuff(2)
	service.DoStuff(3)

	_ = service.Close()

	// Output:
	// 1
	// 2
	// 3
	// received close command, cleaning up
	// sending Done
	// really done, service closed
}
