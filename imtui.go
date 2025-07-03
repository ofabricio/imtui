package imtui

import (
	"iter"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

func New() *ImTui {
	var t ImTui
	t.Style.Text = tcell.StyleDefault.Background(tcell.NewHexColor(0x23272e)).Foreground(tcell.NewHexColor(0xf5f6fa))
	t.Style.Background = tcell.StyleDefault.Background(tcell.NewHexColor(0x23272e)).Foreground(tcell.NewHexColor(0xf5f6fa))
	t.Style.PrimaryAccent = tcell.StyleDefault.Background(tcell.NewHexColor(0x1abc9c)).Foreground(tcell.NewHexColor(0x23272e))
	t.Style.PrimaryAccentLight = tcell.StyleDefault.Background(tcell.NewHexColor(0x48dbb4)).Foreground(tcell.NewHexColor(0x23272e))
	t.Style.SecondaryAccent = tcell.StyleDefault.Background(tcell.NewHexColor(0x34495e)).Foreground(tcell.NewHexColor(0xf5f6fa))
	t.Style.SecondaryAccentLight = tcell.StyleDefault.Background(tcell.NewHexColor(0x2d3a4d)).Foreground(tcell.NewHexColor(0x1abc9c))
	return &t
}

type ImTui struct {
	Style styles
	scrn  tcell.Screen
	mouse mouse
	cur   cursor
}

type styles struct {
	Text                 tcell.Style
	Background           tcell.Style
	PrimaryAccent        tcell.Style
	PrimaryAccentLight   tcell.Style
	SecondaryAccent      tcell.Style
	SecondaryAccentLight tcell.Style
}

func (t *ImTui) Loop() iter.Seq[int] {

	scr, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	t.scrn = scr
	if err := scr.Init(); err != nil {
		panic(err)
	}

	scr.SetStyle(t.Style.Background)

	scr.EnableMouse()

	evtCh := make(chan tcell.Event)
	go scr.ChannelEvents(evtCh, nil)

	return func(yield func(int) bool) {
		var mx, my, lmx, lmy int     // Mouse xy, and last mouse x, y.
		var btn1, lbtn1 bool         // Pressed button1 and last pressed button1.
		var pressed tcell.ButtonMask // Current pressed buttons.
		for {
			select {
			case ev := <-evtCh:
				switch ev := ev.(type) {
				case *tcell.EventResize:
					scr.Sync()

				case *tcell.EventKey:
					switch ev.Key() {
					case tcell.KeyCtrlC, tcell.KeyEscape:
						scr.Fini()
						return
					}

				case *tcell.EventMouse:
					mx, my = ev.Position()
					pressed = ev.Buttons()
					switch pressed {
					case tcell.Button1:
						btn1 = true
					case tcell.ButtonNone:
						btn1 = false
					}
				}
			case <-time.After(60 * time.Millisecond):
			}

			// Begin()
			t.mouse.x = mx
			t.mouse.y = my
			t.mouse.lx, t.mouse.ly = lmx, lmy
			t.mouse.moved = mx != lmx || my != lmy
			t.mouse.pressed = pressed
			t.mouse.btn1 = btn1 != lbtn1
			if t.mouse.IsButton1DownOnce() {
				t.mouse.down = cursor{mx, my}
			}
			t.cur = cursor{}

			scr.Clear()
			if !yield(0) {
				return
			}
			scr.Show()

			// Save current values for tests in the next frames.
			lmx, lmy = mx, my
			lbtn1 = btn1
		}
	}
}

func (t *ImTui) Button(label string) bool {
	v := utf8.RuneCountInString(label)
	r := rect{t.cur.x, t.cur.y, v, 1}
	t.text(label, t.buttonStyle(r))
	return t.mouse.PressedIn(r)
}

func (t *ImTui) Toggle(label string, toggle *bool) bool {
	v := utf8.RuneCountInString(label)
	r := rect{t.cur.x, t.cur.y, v, 1}
	clicked := t.mouse.PressedIn(r)
	if clicked {
		*toggle = !*toggle
	}
	t.text(label, t.toggleStyle(r, *toggle))
	return clicked
}

func (t *ImTui) buttonStyle(r rect) tcell.Style {
	over := t.mouse.In(r)
	down := t.mouse.IsButton1Down()
	switch {
	case over && down:
		return t.Style.PrimaryAccentLight
	case over:
		return t.Style.PrimaryAccent
	default:
		return t.Style.SecondaryAccentLight
	}
}

func (t *ImTui) toggleStyle(r rect, toggled bool) tcell.Style {
	over := t.mouse.In(r)
	down := t.mouse.IsButton1Down()
	switch {
	case over && down || over && toggled:
		return t.Style.PrimaryAccentLight
	case over || toggled:
		return t.Style.PrimaryAccent
	default:
		return t.Style.SecondaryAccentLight
	}
}

func (t *ImTui) Text(text string) bool {
	v := utf8.RuneCountInString(text)
	r := rect{t.cur.x, t.cur.y, v, 1}
	t.text(text, t.Style.Text)
	return t.mouse.PressedIn(r)
}

type widget struct {
	rect    rect
	hover   bool
	clicked bool
}

func (t *ImTui) Move(x, y int) {
	t.cur.x = x
	t.cur.y = y
}

func (t *ImTui) Break() {
	t.cur.x = 0
	t.cur.y++
}

func (t *ImTui) text(text string, style tcell.Style) {
	for _, r := range text {
		t.cur.x++
		t.scrn.SetContent(t.cur.x, t.cur.y, r, nil, style)
	}
}

type cursor struct {
	x, y int
}

type mouse struct {
	x, y    int
	lx, ly  int // Last mouse x, y used to detect mouse movement.
	pressed tcell.ButtonMask
	btn1    bool // Signals changes to Button1.
	moved   bool

	down cursor // Position where the mouse was pressed down.
}

func (m mouse) Entered(r rect) bool {
	return r.Contains(m.x, m.y) && !r.Contains(m.lx, m.ly)
}

func (m mouse) Exited(r rect) bool {
	return !r.Contains(m.x, m.y) && r.Contains(m.lx, m.ly)
}

func (m mouse) IsButton1Down() bool {
	return m.pressed == tcell.Button1
}

func (m mouse) IsButton1DownOnce() bool {
	return m.pressed == tcell.Button1 && m.btn1
}

// IsButton1UpOnce tells if the Button1 was pressed and is now released.
// Returns true only in a single frame.
func (m mouse) IsButton1UpOnce() bool {
	return m.pressed != tcell.Button1 && m.btn1
}

// In tells if the mouse is inside a given rectangle.
func (m mouse) In(r rect) bool {
	return r.Contains(m.x, m.y)
}

// Still tells if the mouse is still.
// A mouse is still if it was pressed down and up in the same place.
func (m mouse) Still() bool {
	return !m.Dragged()
}

// Dragged tells if the mouse was dragged.
// A mouse is dragged if it was pressed down and moved
// to a different position.
func (m mouse) Dragged() bool {
	return m.down.x != m.x || m.down.y != m.y
}

// PressedIn tells if the mouse was pressed down and up
// inside a given rectangle.
func (m mouse) PressedIn(r rect) bool {
	return m.IsButton1UpOnce() && r.Contains(m.x, m.y) && r.Contains(m.down.x, m.down.y)
}

type rect struct {
	x, y int
	w, h int
}

func (r rect) Contains(x, y int) bool {
	return x > r.x && x <= r.x+r.w && y >= r.y && y < r.y+r.h
}
