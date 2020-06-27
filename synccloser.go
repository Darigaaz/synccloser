package synccloser

import (
	"context"
	"errors"
	"fmt"
)

// ErrClosed is the error returned by SyncCloser when closing was already completed.
var ErrClosed = errors.New("already closed")

type Interface interface {
	Close() error
	CloseContext(ctx context.Context) error

	CloseChannel() chan func (err error)
}

type SyncCloser struct {
	closing chan func (err error)
	closed  chan struct{}
}

func New() *SyncCloser {
	return &SyncCloser{
		closing: make(chan func(err error)),
		closed:  make(chan struct{}),
	}
}

func (syncCloser *SyncCloser) CloseChannel() chan func (err error) {
	return syncCloser.closing
}

func (syncCloser *SyncCloser) Close() error {
	return syncCloser.CloseContext(context.Background())
}

func (syncCloser *SyncCloser) CloseContext(ctx context.Context) error {
	// make buffered so slow closers will not block if we didnt wait for it
	errCh := make(chan error, 1)
	doneFunc := func (err error) {
		close(syncCloser.closed)

		errCh <- err
		close(errCh)
	}

	// @note We dont close closing channel, no need to pre-check
	//select {
	//case <-ctx.Done():
	//	return fmt.Errorf("timeout waiting for sending close command: %w", ctx.Err())
	//case <-syncCloser.closed:
	//	fmt.Println("already closed!!!")
	//	return ErrClosed
	//
	//default:
	//}

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for sending close command: %w", ctx.Err())

	case <-syncCloser.closed:
		//fmt.Println("closed while sending close command")
		return ErrClosed

	case syncCloser.closing <- doneFunc:
		select {
		case <-ctx.Done():
			// @todo return comparable error (err.Is(sync_closer.TimeoutClosing) == true)
			return fmt.Errorf("timeout waiting for Close to complete: %w", ctx.Err())

		case err := <-errCh:
			//fmt.Println("closed successfully")
			return err
		}
	}
}
