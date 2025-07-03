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
					t.mouse.x, t.mouse.y = mx, my
					t.mouse.moved = t.mouse.x != t.mouse.lx || t.mouse.y != t.mouse.ly
					t.mouse.pressed = pressed
					if t.mouse.IsButton1DownOnce() {
						t.mouse.down = cursor{t.mouse.x, t.mouse.y}
					}
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

			// Save current values for tests in the next frames.
			t.mouse.lx, t.mouse.ly = t.mouse.x, t.mouse.y
			t.mouse.lpressed = t.mouse.pressed
		}
	}
}

// Button creates a button with the given label.
// Returns true if the button was clicked.
func (t *ImTui) Button(label string) bool {
	r := t.textRect(label)
	t.text(label, t.buttonStyle(r))
	return t.mouse.PressedIn(r)
}

// Toggle creates a toggle button with the given label.
// The toggle is a boolean pointer that will be toggled when the button is clicked.
// Returns true if the toggle was clicked.
func (t *ImTui) Toggle(label string, toggle *bool) bool {
	r := t.textRect(label)
	clicked := t.toggle(r, toggle)
	t.text(label, t.toggleStyle(r, *toggle))
	return clicked
}

// Check creates a checkbox with the given label.
// The toggle is a boolean pointer that will be toggled when the checkbox is clicked.
// Returns true if the checkbox was clicked.
func (t *ImTui) Check(label string, toggle *bool) bool {
	check := "[ ] "
	r := t.textRect(label)
	r.w += len(check)
	clicked := t.toggle(r, toggle)
	if *toggle {
		check = "[X] "
	}
	s := t.toggleStyle(r, *toggle)
	t.text(check, s)
	t.text(label, s)
	return clicked
}

// Radio creates a radio button with the given label.
// The id is the unique identifier for the radio button.
// The radio is set to the id when the radio button is clicked.
// Returns true if the radio button was clicked.
func (t *ImTui) Radio(label string, id int, radio *int) bool {
	toggle := *radio == id
	if t.Check(label, &toggle) {
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
	r := t.textRect(text)
	t.text(text, t.Style.Text)
	return t.mouse.PressedIn(r)
}

func (t *ImTui) toggle(r rect, toggle *bool) bool {
	clicked := t.mouse.PressedIn(r)
	if clicked {
		*toggle = !*toggle
	}
	return clicked
}

func (t *ImTui) textRect(text string) rect {
	return rect{t.cur.x, t.cur.y, t.chars(text), 1}
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

func (t *ImTui) chars(text string) int {
	return utf8.RuneCountInString(text)
}

type widget struct {
	rect    rect
	hover   bool
	clicked bool
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
	x, y   int
	lx, ly int // Last mouse x, y used to detect mouse movement.

	moved bool // Tells if the mouse was moved in the last frame.

	pressed  tcell.ButtonMask // Current mouse button pressed.
	lpressed tcell.ButtonMask // Last mouse button pressed used to detect mouse button changes.

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
	return m.pressed == tcell.Button1 && m.IsButtonChanged()
}

// IsButton1UpOnce tells if the Button1 was pressed and is now released.
// Returns true only in a single frame.
func (m mouse) IsButton1UpOnce() bool {
	return m.pressed != tcell.Button1 && m.IsButtonChanged()
}

// IsButtonChanged tells if the mouse button state changed.
// Returns true only in a single frame.
func (m mouse) IsButtonChanged() bool {
	return m.lpressed != m.pressed
}

// Dragged tells if the mouse was dragged.
// A mouse is dragged if it was pressed down and moved
// to a different position.
func (m mouse) Dragged() bool {
	return m.down.x != m.x || m.down.y != m.y
}

// In tells if the mouse is inside a given rectangle.
func (m mouse) In(r rect) bool {
	return r.Contains(m.x, m.y)
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
