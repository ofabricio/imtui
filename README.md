# imtui

Simple Immediate Mode Terminal UI

## Example

```go
package main

import "github.com/ofabricio/imtui"

func main() {

    tui := imtui.New()

    var v int
    for range tui.Loop() {

        if tui.Button(" Click! ") {
            v++
        }

        tui.Text(fmt.Sprintf(" Button clicked %d times ", v))
    }
}
```

![Output](/.github/output.png)
