package alp

import "os"
import "fmt"

type World struct {
	dir string
	obj map[oid]*Obj
	max oid
}

func (w *World) makeobj(id oid) *Obj {
	o := &Obj{w: w, Id: id}
	w.obj[id] = o
	o.from.Init()
	o.to.Init()
	if id > w.max {
		w.max = id
	}
	return o
}

func (w *World) Load(id oid) (o *Obj, err error) {
	o = w.makeobj(id)
	if err = o.Load(id); err != nil {
		delete(w.obj, id)
	}
	return
}

func (w *World) Linkadd(o *Obj, domid oid, codid oid) error {
	if domid == 0 && codid == 0 {
		return nil
	}
	dom := w.obj[domid]
	cod := w.obj[codid]
	if dom == nil || cod == nil {
		return fmt.Errorf("Unknown link end: %s or %s in %s",
			domid, codid, o.Id)
	}
	o.Link(dom, cod)
	return nil
}

func (w *World) CreateObj() (o *Obj) {
	o = w.makeobj(w.max + 1)
	o.check()
	return
}

func (w *World) Store(os []*Obj) (err error) {
	for _, o := range os {
		err  = o.Store()
		if err != nil {
			break
		}
	}
	return
}

func (w *World) Open(dir string) (err error) {
	var fd *os.File

	w.dir = dir + "/o/"
	fd, err = os.Open(w.dir)
	if err == nil {
		defer fd.Close()
		var names []string
		names, err = fd.Readdirnames(0)
		if err == nil {
			w.obj = make(map[oid]*Obj, len(names))
			for _, name := range names {
				var id oid

				if id, err = strid(name); err == nil {
					_, err = w.Load(id)
				}
				if err != nil {
					break
				}
			}
		}
	}
	return
}

func (w *World) Print() {
	for _, o := range w.obj {
		fmt.Println(o)
	}
}

