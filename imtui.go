package imtui

import (
	"iter"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/gdamore/tcell/v2"
)

func New() ImTui {
	var t ImTui
	t.Style.Text = tcell.StyleDefault.Background(tcell.NewHexColor(0x181818)).Foreground(tcell.NewHexColor(0xffffff))
	t.Style.Background = tcell.StyleDefault.Background(tcell.NewHexColor(0x181818)).Foreground(tcell.NewHexColor(0xffffff))
	t.Style.Active = tcell.StyleDefault.Background(tcell.NewHexColor(0x264f78)).Foreground(tcell.NewHexColor(0xffffff))
	t.Style.OverActive = tcell.StyleDefault.Background(tcell.NewHexColor(0x1565c0)).Foreground(tcell.NewHexColor(0xffffff))
	t.Style.Over = tcell.StyleDefault.Background(tcell.NewHexColor(0x1976d2)).Foreground(tcell.NewHexColor(0xffffff))
	t.Style.Normal = tcell.StyleDefault.Background(tcell.NewHexColor(0x23272e)).Foreground(tcell.NewHexColor(0xffffff))
	return t
}

type ImTui struct {
	Style styles
	scrn  tcell.Screen
	mouse mouse
	cur   cursor

	// NOTE: Maybe this should go to mouse struct? If so, ImTui.mouseStates() can become mouse.states().
	active delta[int]
}

type styles struct {
	Text       tcell.Style
	Background tcell.Style
	Normal     tcell.Style
	Active     tcell.Style
	Over       tcell.Style
	OverActive tcell.Style
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

		t.mouse.x.curr = -1
		t.mouse.y.curr = -1

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
					mx, my := ev.Position()
					pressed := ev.Buttons()
					t.mouse.Update(mx, my, pressed)
				}
			case <-time.After(60 * time.Millisecond):
			}

			t.cur = cursor{}

			scr.Clear()
			if !yield(0) {
				scr.Fini()
				return
			}
			scr.Show()

			// Make current values as last values
			// for change test in the next frame.
			t.mouse.x.Swap()
			t.mouse.y.Swap()
			t.mouse.pressed.Swap()
			t.active.Swap()
		}
	}
}

// Button creates a button with the given label.
// Returns true if the button was clicked.
func (t *ImTui) Button(label string) bool {
	a := t.textArea(label)
	s := t.mouseStates(label, a)
	t.fillText(label, t.buttonStyle(s))
	return s.clicked
}

// Toggle creates a toggle button with the given label.
// The toggle is a boolean pointer that will be toggled when the button is clicked.
// Returns true if the toggle was clicked.
func (t *ImTui) Toggle(label string, toggle *bool) bool {
	return t.Toggler("", "", label, toggle)
}

// Check creates a checkbox with the given label.
// The toggle is a boolean pointer that will be toggled when the checkbox is clicked.
// Returns true if the checkbox was clicked.
func (t *ImTui) Check(label string, toggle *bool) bool {
	return t.Toggler("[ ] ", "[X] ", label, toggle)
}

// Radio creates a radio button with the given label.
// The id is the unique identifier for the radio button.
// The radio is set to the id when the radio button is clicked.
// Returns true if the radio button was clicked.
func (t *ImTui) Radio(label string, id int, radio *int) bool {
	toggle := *radio == id
	if t.Toggler("( ) ", "(0) ", label, &toggle) {
		if toggle {
			*radio = id
		} else {
			*radio = -1
		}
		return true
	}
	return false
}

// Text creates a text label with the given text.
// Returns true if the mouse was pressed inside the text area.
func (t *ImTui) Text(text string) bool {
	a := t.textArea(text)
	t.fillText(text, t.Style.Text)
	return t.mouse.PressedIn(a)
}

// Toggler creates a toggle button that allows further customization.
func (t *ImTui) Toggler(off, on, label string, toggle *bool) bool {
	a := t.textArea(label)
	a.x2 += len(off)
	s := t.mouseStates(label, a)
	if s.clicked {
		*toggle = !*toggle
	}
	check := off
	if *toggle {
		check = on
	}
	style := t.toggleStyle(s, *toggle)
	t.fillText(check, style)
	t.fillText(label, style)
	return s.clicked
}

func (t *ImTui) textArea(text string) area {
	return area{t.cur.x, t.cur.y, t.cur.x + t.chars(text) - 1, t.cur.y}
}

func (t *ImTui) buttonStyle(s mouseStates) tcell.Style {
	switch {
	case s.down:
		return t.Style.Active
	case s.over:
		return t.Style.Over
	default:
		return t.Style.Normal
	}
}

func (t *ImTui) toggleStyle(s mouseStates, toggled bool) tcell.Style {
	switch {
	case s.down || toggled && !s.over:
		return t.Style.Active
	case toggled && s.over:
		return t.Style.OverActive
	case s.over:
		return t.Style.Over
	default:
		return t.Style.Normal
	}
}

// Move moves the cursor to a given position.
func (t *ImTui) Move(x, y int) {
	t.cur.x = x
	t.cur.y = y
}

// MoveRel moves the cursor relative to its current position.
func (t *ImTui) MoveRel(x, y int) {
	t.cur.x += x
	t.cur.y += y
}

// Break moves the cursor to the next line and resets the x position.
func (t *ImTui) Break() {
	t.cur.x = 0
	t.cur.y++
}

func (t *ImTui) Size() (w, h int) {
	return t.scrn.Size()
}

func (t *ImTui) mouseStates(id string, a area) mouseStates {
	var s mouseStates
	if s.over = t.mouse.In(a); s.over {
		t.active.curr = strID(id)
		s.active = !t.active.Changed()
		s.clicked = s.active && t.mouse.PressedIn(a)
		s.down = s.active && t.mouse.IsButton1Down()
		s.over = s.active && s.over
	}
	return s
}

func (t *ImTui) fillText(text string, s tcell.Style) {
	for _, r := range text {
		t.scrn.SetContent(t.cur.x, t.cur.y, r, nil, s)
		t.cur.x++
	}
}

func (t *ImTui) fill(a area, s tcell.Style) {
	for y := a.y1; y < a.y2; y++ {
		for x := a.x1; x < a.x2; x++ {
			t.scrn.SetContent(x, y, ' ', nil, s)
		}
	}
}

func (t *ImTui) chars(text string) int {
	return utf8.RuneCountInString(text)
}

type cursor struct {
	x, y int
}

type mouse struct {
	x, y    delta[int]
	pressed delta[tcell.ButtonMask]

	down cursor // Position where the mouse was pressed down.
}

func (m *mouse) Update(x, y int, pressed tcell.ButtonMask) {
	m.x.curr, m.y.curr = x, y
	m.pressed.curr = pressed
	if m.IsButton1DownOnce() {
		m.down = cursor{x, y}
	}
}

// Entered tells if the mouse entered the area.
func (m mouse) Entered(a area) bool {
	return a.Contains(m.x.curr, m.y.curr) && !a.Contains(m.x.last, m.y.last)
}

// Exited tells if the mouse exited the area.
func (m mouse) Exited(a area) bool {
	return !a.Contains(m.x.curr, m.y.curr) && a.Contains(m.x.last, m.y.last)
}

// IsButton1Down tells if the mouse is down,
// which may happen in many frames.
func (m mouse) IsButton1Down() bool {
	return m.pressed.curr == tcell.Button1
}

// IsButton1DownOnce tells if the mouse is down
// just in a single frame.
func (m mouse) IsButton1DownOnce() bool {
	return m.pressed.curr == tcell.Button1 && m.IsButtonChanged()
}

// IsButton1UpOnce tells if the Button1 was pressed and is now released.
// Returns true only in a single frame.
func (m mouse) IsButton1UpOnce() bool {
	return m.pressed.curr != tcell.Button1 && m.IsButtonChanged()
}

// IsButtonChanged tells if the mouse button state changed.
// Returns true only in a single frame.
func (m mouse) IsButtonChanged() bool {
	return m.pressed.Changed()
}

// Dragged tells if the mouse was dragged.
// A mouse is dragged if it was pressed down and moved
// to a different position.
func (m mouse) Dragged() bool {
	return m.down.x != m.x.curr || m.down.y != m.y.curr
}

// In tells if the mouse is inside a given area.
func (m mouse) In(a area) bool {
	return a.Contains(m.x.curr, m.y.curr)
}

// PressedIn tells if the mouse was pressed down and up
// inside a given area.
func (m mouse) PressedIn(a area) bool {
	return m.IsButton1UpOnce() && a.Contains(m.x.curr, m.y.curr) && a.Contains(m.down.x, m.down.y)
}

type area struct {
	x1, y1 int
	x2, y2 int
}

func (a area) Contains(x, y int) bool {
	return x >= a.x1 && x <= a.x2 && y >= a.y1 && y <= a.y2
}

// type dragdrop struct {
// 	down, up cursor
// }

// delta represents a value that may change in each frame.
type delta[T comparable] struct {
	curr T
	last T
}

// Changed tells if the delta value changed.
// In case of a change returns true in a single frame.
func (d delta[T]) Changed() bool {
	return d.curr != d.last
}

// Swap swaps the last value with the current value.
func (d *delta[T]) Swap() {
	d.last = d.curr
}

type mouseStates struct {
	clicked bool
	active  bool
	over    bool
	down    bool
}

// strID returns the address of the string data as an ID.
// This means that two strings with the same content will
// have the same ID, unless they point to different memory.
func strID(s string) int {
	return int(uintptr(unsafe.Pointer(unsafe.StringData(s))))
}
