package closer

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var globalCloser = New(syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

type Closer struct {
	mu    sync.Mutex
	once  sync.Once
	done  chan struct{}
	funcs []func() error
}

// New returns new Closer, If list os.Signal are specified Closer will automatically call CloseAll when one of signals is received from OS
func New(sig ...os.Signal) *Closer {
	c := &Closer{
		done: make(chan struct{}),
	}

	if len(sig) > 0 {
		go func() {
			ch := make(chan os.Signal, 1)

			signal.Notify(ch, sig...)
			<-ch
			signal.Stop(ch)

			c.CloseAll()
		}()
	}

	return c
}

// Add func to closer
func Add(fn ...func() error) {
	globalCloser.Add(fn...)
}

// Wait blocks until all closer functions are done
func Wait() {
	globalCloser.Wait()
}

// CloseAll calls all closer functions
func CloseAll() {
	globalCloser.CloseAll()
}

func (c *Closer) Add(fn ...func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, fn...)
}

func (c *Closer) Wait() {
	<-c.done
}

func (c *Closer) CloseAll() {
	c.once.Do(func() {
		defer close(c.done)

		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		errs := make(chan error, len(funcs))

		for _, fn := range funcs {
			go func(fn func() error) {
				errs <- fn()
			}(fn)
		}

		for i := 0; i < cap(errs); i++ {
			if err := <-errs; err != nil {
				logger.Error("error returned from closer", "error", err)
			}
		}

		logger.Info("all closer functions are done, shutting down the app")
	})
}
