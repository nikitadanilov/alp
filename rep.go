package alp

import "container/list"
import "strings"
import "strconv"
import "errors"
import "bufio"
import "fmt"
import "os"

type Rep struct {
	history list.List
	cur     *list.Element
	w       *World
	dirty   dirtylist
}

func (r *Rep) Cur() (o *Obj) {
	return cast(r.cur)
}

func (r *Rep) prompt() (s string) {
	s = r.Cur().Name
	if len(s) == 0 {
		s = "#" + r.Cur().Id.String()
	}
	return s + "; "
}

type command struct {
	name string
	f    func (r *Rep, arg []string) error
	min  int
	max  int
	msg  string
}

var commands []command

func (r *Rep) Init(w *World, id oid) error {
	commands = []command{
		{"p", (*Rep).print, 0, 1000000, "Print object (. by default)" },
		{"P", (*Rep).printR, 0, 1000000, 
			"Print object raw (. by default)" },
		{"?", (*Rep).help,  0, 0, "Help message" },
		{"g", (*Rep).go_to,  1, 1, "Goto" },
		{"<", (*Rep).backward,  0, 1, "Backward" },
		{">", (*Rep).forward,  0, 1, "Forward" },
		{"h", (*Rep).showHistory,  0, 0, "Print history" },
		{"n", (*Rep).name,  1, 1000000, "Set name" },
		{"+", (*Rep).newobj, 0, 0, "New object" },
		{"l", (*Rep).link, 2, 2, "Link" },
		{"t", (*Rep).linkto, 1, 2, "Link from" },
		{"f", (*Rep).linkfrom, 1, 2, "Link to" },
	}
	r.w = w
	o := w.obj[1]
	if o == nil {
		return errors.New(fmt.Sprintf("No object %s", id.String()))
	}
	r.history.Init()
	r.cur = r.history.PushBack(o)
	return nil
}

func (r *Rep) do() bool {
	var line string
	var arg  []string
	var err  error

	fmt.Print(r.prompt())
	if line, err = bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		return false
	}
	line = line[0 : len(line) - 1]
	if len(line) == 0 {
		return true
	}
	arg = strings.Split(line, " ")
	for _, cmd := range commands {
		if cmd.name == arg[0] {
			if len(arg) < cmd.min + 1 {
				r.error("%s requires at least %d arguments",
					cmd.name, cmd.min)
				return true
			}
			if len(arg) > cmd.max + 1 {
				r.error("%s requires at most %d arguments",
					cmd.name, cmd.max)
				return true
			}
			err = cmd.f(r, arg[1:])
			if err == nil {
				err = r.dirty.clean(r.w)
			}
			if err != nil {
				r.error("%v", err)
			}
			return true
		}
	}
	r.error("Unrecognized command %q", arg[0])
	return true
}

func (r *Rep) Loop() (err error) {
	for r.do() {
		;
	}
	return
}

func (r *Rep) error(msg string, a ...interface{}) error {
	msg = fmt.Sprintf(msg, a...)
	fmt.Fprintln(os.Stderr, msg)
	return errors.New(msg)
}

func (r *Rep) Obj(name string) (o *Obj, err error) {
	var id oid

	if name == "." {
		o = r.Cur()
	} else if id, err = strid(name); err == nil {
		o = r.w.obj[id]
		if o == nil {
			err = fmt.Errorf("Cannot find object %s", name)
		}
	} 
	return
}

func (r *Rep) printAs(arg []string, f func (o *Obj) string) (err error) {
	var o *Obj

	if len(arg) == 0 {
		arg = []string{"."}
	}
	for _, a := range arg {
		o, err = r.Obj(a)
		if err != nil {
			return err
		}
		fmt.Print(f(o))
	}
	return nil
}

func (r *Rep) print(arg []string) (err error) {
	return r.printAs(arg, (*Obj).Print)
}

func (r *Rep) printR(arg []string) (err error) {
	return r.printAs(arg, (*Obj).String)
}

func (r *Rep) help(arg []string) error {
	for _, cmd := range commands {
		fmt.Printf("%q: %s. Takes between %d and %d arguments\n",
			cmd.name, cmd.msg, cmd.min, cmd.max)
	}
	return nil
}

func (r *Rep) go_to(arg []string) (err error) {
	o, err := r.Obj(arg[0])
	if err == nil {
		r.cur = r.history.PushBack(o)
	}
	return
}

func (r *Rep) backward(arg []string) (err error) {
	nr, err := strint(arg, 1)
	if err != nil {
		return
	}
	for i := 0; i < nr; i++ {
		prev := r.cur.Prev()
		if prev != nil {
			r.cur = prev
		}
	}
	return
}

func (r *Rep) forward(arg []string) (err error) {
	nr, err := strint(arg, 1)
	if err != nil {
		return
	}
	for i := 0; i < nr; i++ {
		next := r.cur.Next()
		if next != nil {
			r.cur = next
		}
	}
	return
}

func (r *Rep) showHistory(arg []string) (err error) {
	for e := r.history.Front(); e != nil; e = e.Next() {
		var s string

		if e == r.cur {
			s = "* "
		} else {
			s = "  "
		}
		o := cast(e)
		fmt.Printf("%v %v %v\n", s, o.Name, o.Id)
	}
	return nil
}

func (r *Rep) name(arg []string) (err error) {
	r.Cur().Name = strings.Join(arg, " ")
	r.dirty.add(r.Cur())
	return
}

func (r *Rep) newobj(arg []string) (err error) {
	r.cur = r.history.PushBack(r.w.CreateObj())
	r.dirty.add(r.Cur())
	return
}

func (r *Rep) link(arg []string) (err error) {
	dom, err := r.Obj(arg[0])
	if err != nil {
		return
	}
	cod, err := r.Obj(arg[1])
	if err != nil {
		return
	}
	r.Cur().Link(dom, cod)
	r.dirty.add(r.Cur())
	return
}

func (r *Rep) linkbuild(arg []string) (o *Obj, link *Obj, err error) {
	o, err = r.Obj(arg[0])
	link = r.w.CreateObj()
	if len(arg) > 1 {
		link.Name = arg[1]
	}
	r.dirty.add(r.Cur())
	r.dirty.add(o)
	r.dirty.add(link)
	return
}

func (r *Rep) linkto(arg []string) (err error) {
	o, link, err := r.linkbuild(arg)
	link.Link(o, r.Cur());
	return
}

func (r *Rep) linkfrom(arg []string) (err error) {
	o, link, err := r.linkbuild(arg)
	link.Link(r.Cur(), o);
	return
}

const maxDirty = 16

type dirtylist struct {
	used    int
	laundry [maxDirty]*Obj
}

func (d *dirtylist) add(o *Obj) {
	if d.used == maxDirty {
		panic("Too many dirty objects.");
	}
	d.laundry[d.used] = o
	d.used++
}

func (d *dirtylist) clean(w *World) (err error) {
	return w.Store(d.laundry[0 : d.used])
}

func strint(s []string, def int) (nr int, err error) {
	if len(s) == 0 {
		nr = def
	} else {
		nr, err = strconv.Atoi(s[0])
	}
	return
}
