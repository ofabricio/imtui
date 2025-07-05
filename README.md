# imtui

Simple Immediate Mode Terminal UI

## Example

```go
package main

import "github.com/ofabricio/imtui"

func main() {

    tui := imtui.New()

    var clicks int
    for range tui.Loop() {

        if tui.Button(" Click! ") {
            clicks++
        }

        tui.Text(fmt.Sprintf(" Button clicked %d times ", clicks))
    }
}
```

## Demo

This demo is from the example in [example/example.go](example/demo.go).

<p align="center">
  <img src="/.github/demo.gif" />
</p>

## Documentation

### Text

```go
for range tui.Loop() {

    tui.Text("Hello, World!")
}
```

### Button

```go
var clicks int
for range tui.Loop() {

    if tui.Button(" Click! ") {
        clicks++
    }

    tui.Text(fmt.Sprintf(" Button clicked %d times ", clicks))
}
```

### Toggle Button

```go
var toggle bool
for range tui.Loop() {

    if tui.Toggle(" Expand ", &toggle); toggle {
        tui.Text(" Hello! ")
    }
}
```

### Checkbox

```go
var one, two bool
for range tui.Loop() {

    tui.Check("One", &one)
    tui.Check("Two", &two)

    tui.Text(fmt.Sprintf(" One is %t; Two is %t ", one, two))
}
```

### Radio button

```go
var opt int = -1
for range tui.Loop() {

    tui.Radio("One", 0, &opt)
    tui.Radio("Two", 1, &opt)

    tui.Text(fmt.Sprintf(" Item selected: %v ", opt))
}
```
