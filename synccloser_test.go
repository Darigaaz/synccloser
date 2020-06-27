package synccloser_test

import (
	"errors"
	"testing"

	"github.com/Darigaaz/synccloser"
)

type TestService struct {
	// embed SyncCloser
	synccloser.SyncCloser

	C chan int
}

func NewTestService() *TestService {
	return &TestService{
		// instantiate SyncCloser
		// currently its not possible to use zero value
		// because we need to make channels in race free manner
		SyncCloser: *synccloser.New(),

		C: make(chan int),
	}
}

func (service *TestService) Start(cleanup func() error) {
	if cleanup == nil {
		cleanup = func() error {return nil }
	}

	var lastError error
	closeChannel := service.CloseChannel()

	for {
		select {
		case done := <-closeChannel:
			lastError = cleanup()
			done(lastError)

			return
		}
	}
}

func (service *TestService) cleanup() error {
	// cleanup logic goes here

	return nil
}

// You can omit this method
func (service *TestService) Close() error {
	// custom logic goes here

	return service.SyncCloser.Close()
}

func TestSyncCloser_Close(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		service := NewTestService()

		go service.Start(nil)

		err := service.Close()
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
	})

	t.Run("multiple calls to Close", func(t *testing.T) {
		service := NewTestService()

		go service.Start(nil)

		errs := []error{service.Close(), service.Close(), service.Close()}

		if !(errors.Is(errs[0], nil) && errors.Is(errs[1], synccloser.ErrClosed) && errors.Is(errs[2], synccloser.ErrClosed)) {
			t.Errorf(`Expected no [<nil> "already closed" "already closed"], got %q`, errs)
		}
	})

	t.Run("premature calls to Close", func(t *testing.T) {
		service := NewTestService()

		var errs []error
		ch := make(chan error)
		for i := 0; i < 3; i++ {
			go func() {
				ch <- service.Close()
			}()
		}

		// blocking call
		service.Start(nil)

		for i := 0; i < 3; i++ {
			errs = append(errs, <- ch)
		}

		nilErrs := 0
		errClosedErrs := 0
		for _, err := range errs {
			switch err {
			case nil:
				nilErrs++
			case synccloser.ErrClosed:
				errClosedErrs++
			default:
				t.Errorf("Expected only nil and 'already closed' errors, got: %q", err)
			}
		}

		if !(nilErrs == 1 && errClosedErrs == 2) {
			t.Errorf(`Expected one no error and two "already closed", got %q`, errs)
		}
	})


	t.Run("long cleanup", func(t *testing.T) {
		service := NewTestService()
		block := make(chan error)

		go service.Start(func() error { return <-block })

		var errs []error
		ch := make(chan error)
		for i := 0; i < 3; i++ {
			go func() {
				ch <- service.Close()
			}()
		}

		// unblock close once
		block <- nil

		for i := 0; i < 3; i++ {
			errs = append(errs, <- ch)
		}

		nilErrs := 0
		errClosedErrs := 0
		for _, err := range errs {
			switch err {
			case nil:
				nilErrs++
			case synccloser.ErrClosed:
				errClosedErrs++
			default:
				t.Errorf("Expected only nil and 'already closed' errors, got: %q", err)
			}
		}

		if !(nilErrs == 1 && errClosedErrs == 2) {
			t.Errorf(`Expected one no error and two "already closed", got %q`, errs)
		}
	})
}
