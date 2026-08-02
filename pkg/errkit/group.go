package errkit

import (
	"sync"
)

type Group struct {
	mutex   sync.Mutex
	err     *Error
	finally []func() error
	wg      sync.WaitGroup
}

func (g *Group) Add(err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if err == nil {
		return
	}
	g.err = Append(g.err, err)
}

func (g *Group) Go(f func() error) {
	g.wg.Go(func() {
		if err := f(); err != nil {
			g.Add(err)
		}
	})
}

func (g *Group) Finally(f func() error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.finally = append(g.finally, f)
}

func (g *Group) Wait() error {
	g.wg.Wait()

	g.mutex.Lock()
	funcs := g.finally
	g.finally = nil
	g.mutex.Unlock()
	for _, f := range funcs {
		if err := f(); err != nil {
			g.Add(err)
		}
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.err == nil {
		return nil
	}
	return g.err
}
