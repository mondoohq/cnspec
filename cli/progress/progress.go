package progress

import (
	"sync"
	"sync/atomic"
)

type Progress interface {
	Open()
	OnProgress(current int, total int)
	Close()
}

type Noop struct{}

func (n Noop) Open()               {}
func (n Noop) OnProgress(int, int) {}
func (n Noop) Close()              {}

type ProgressData struct {
	SortedKeys []string
	Names      map[string]string
	Progress   sync.Map
	complete   int32
}

func (p *ProgressData) IsComplete() bool {
	return atomic.LoadInt32(&p.complete) > 0
}

func (p *ProgressData) SetComplete() {
	atomic.StoreInt32(&p.complete, 1)
}

type progressbar struct {
	id           string
	completion   uint32
	maxNameWidth int
	Data         *ProgressData
}

func New(id string, name string) *progressbar {
	return NewMultiBar(id, &ProgressData{
		SortedKeys: []string{name},
		Names:      map[string]string{id: name},
		Progress:   sync.Map{},
	})
}

func NewMultiBar(id string, data *ProgressData) *progressbar {
	maxNameWidth := 0
	for _, v := range data.Names {
		l := len(v)
		if l > maxNameWidth {
			maxNameWidth = l
		}
	}

	return &progressbar{
		id:           id,
		completion:   0,
		maxNameWidth: maxNameWidth,
		Data:         data,
	}
}

func (p *progressbar) Open() {
	panic("progressbar.Open not yet implemented")
}

func (p *progressbar) OnProgress(current int, total int) {
	panic("progressbar.OnProgress not yet implemented")
}

func (p *progressbar) Close() {
	panic("progressbar.Close not yet implemented")
}
