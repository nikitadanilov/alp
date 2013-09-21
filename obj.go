package alp

import "container/list"
import "errors"
import "bytes"
import "fmt"
import "os"

type Obj struct {
	w     *World
	Id    oid
	from  list.List
	to    list.List
	dom   *Obj
	cod   *Obj
	src   *list.Element
	tgt   *list.Element
	Name  string
	Notes string
}

func (o *Obj) String() string {
	var buf bytes.Buffer

	badd(&buf, "[\"%s\" (%v): %s]", o.Name, o.Id, o.Notes)
	if o.IsLink() {
		badd(&buf, "\n\t%s -> %s", o.dom.Name, o.cod.Name)
	}
	for e := o.to.Front(); e != nil; e = e.Next() {
		badd(&buf, "\n\t-> %s", e.Value.(*Obj).Name)
	}
	for e := o.from.Front(); e != nil; e = e.Next() {
		badd(&buf, "\n\t<- %s", e.Value.(*Obj).Name)
	}
	return buf.String()
}

const (
	toList   = 0
	fromList = 1
)

func (o *Obj) Print() string {
	var buf bytes.Buffer
	var cnt int

	o.printList(toList, &cnt, &buf)
	badd(&buf, "[\"%s\" (%v): %s]", o.Name, o.Id, o.Notes)
	if o.IsLink() {
		badd(&buf, " %s -> %s", o.dom.Name, o.cod.Name)
	}
	badd(&buf, "\n")
	o.printList(fromList, &cnt, &buf)
	return buf.String()
}

func (o *Obj) IsLink() bool {
	return o.dom != nil
}

func (o *Obj) invariant() bool {
	return o.w != nil && o.Id > 0 && o.w.obj[o.Id] == o && o.Id <= o.w.max &&
		!o.IsLink() || (o.cod != nil &&
			o.src != nil && o.tgt != nil &&
			o.src.Value == o && o.tgt.Value == o)
}

func (o *Obj) check() {
	if !o.invariant() {
		panic(o)
	}
	return
}

func (o *Obj) dir() string {
	return o.w.dir + o.Id.String()
}

func (o *Obj) Link(dom *Obj, cod *Obj) {
	o.check()
	o.dom = dom
	o.cod = cod
	o.src = o.dom.from.PushBack(o)
	o.tgt = o.cod.to.PushBack(o)
	o.check()
}

func (o *Obj) Load(id oid) (err error) {
	var (
		odir    string
		fd      *os.File
		domid   oid
		codid   oid
	)
	odir = o.dir()
	fd, err = os.Open(odir + "/.meta")
	if err != nil {
		return
	}
	defer func() { if fd != nil { fd.Close() } }()

	_, err = fmt.Fscanf(fd, "%08x\n%q\n%08x\n%08x", 
		&o.Id, &o.Name, &domid, &codid)
	if err != nil {
		return
	}
	if o.Id != id {
		  return errors.New(fmt.Sprintf("id mismatch: %v != %v",
			id, o.Id))
	}
	if err = o.w.Linkadd(o, domid, codid); err != nil {
		return
	}
	fd.Close()
	fd, err = os.Open(odir + "/.notes")
	if err == nil {
		_, err = fmt.Fscanf(fd, "%q", &o.Notes)
	} else {
		err = nil
	}
	o.check()
	return
}

func (o *Obj) Store() (err error) {
	var (
		fd    *os.File
		domid oid
		codid oid
	)
	o.check()
	err = os.MkdirAll(o.dir(), 0700)
	if err != nil {
		return
	}
	fd, err = os.Create(o.dir() + "/.meta")
	if err != nil {
		return
	}
	if o.IsLink() {
		domid = o.dom.Id
		codid = o.cod.Id
	}
	_, err = fmt.Fprintf(fd, "%v\n%q\n%v\n%v", o.Id, o.Name, domid, codid)
	if err != nil {
		return
	}
	fd.Close()
	if len(o.Notes) > 0 {
		fd, err = os.Create(o.dir() + "/.notes")
		if err != nil {
			return
		}
		fmt.Fprintf(fd, "%q", o.Notes)
		fd.Close()
	}
	o.check()
	return
}

func (o *Obj) printList(direction int, cnt *int, buf *bytes.Buffer) {
	var l *list.List

	if direction == toList {
		l = &o.to
	} else {
		l = &o.from
	}
	for e := l.Front(); e != nil; e = e.Next() {
		var rel, other *Obj

		rel = cast(e)
		if direction == toList {
			other = rel.dom
		} else {
			other = rel.cod
		}
		badd(buf, "   %3x %8.8s | %s\n", *cnt, rel.Name, other.Name)
		*cnt++
	}
}

type oid uint64

func strid(s string) (id oid, err error) {
	_, err = fmt.Sscanf(s, "%x", &id)
	return
}

func (id oid) String() string {
	return fmt.Sprintf("%08.8x", uint64(id))
}

func badd(buf *bytes.Buffer, format string, args ...interface{}) {
	buf.WriteString(fmt.Sprintf(format, args...))
}

func cast(e *list.Element) *Obj {
	o, ok := e.Value.(*Obj)
	if !ok {
		panic(fmt.Sprintf("Non-object in list: %T", e.Value))
	}
	o.check()
	return o
}

