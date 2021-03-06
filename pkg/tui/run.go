package tui

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/dytlzl/tervi/pkg/key"
)

func Print(createView func() *View, options ...option) error {
	cfg := config{}
	for _, opt := range options {
		err := opt(&cfg)
		if err != nil {
			return err
		}
	}

	isAlternative := false

	r, err := newRenderer(isAlternative)
	if err != nil {
		return fmt.Errorf("failed to init renderer: %w", err)
	}
	defer func() {
		r.close(isAlternative)
		for _, obj := range bufferForDebug {
			fmt.Printf("%#v\n", obj)
		}
	}()

	if changed, _ := r.updateTerminalSize(); changed {
		r.shouldSkipRendering = false
	}

	// Clear
	r.fill(style{})

	v := ZStack(createView())

	// Render views
	err = renderView(r, v, &cfg, rect{0, 0, r.width, r.height}, style{})
	if err != nil {
		return fmt.Errorf("failed to render view: %w", err)
	}

	// Draw
	r.draw()

	return nil
}

func Run(createView func() *View, options ...option) error {
	cfg := config{}
	for _, opt := range options {
		err := opt(&cfg)
		if err != nil {
			return err
		}
	}

	isAlternative := true

	r, err := newRenderer(isAlternative)
	if err != nil {
		return fmt.Errorf("failed to init renderer: %w", err)
	}
	defer func() {
		r.close(isAlternative)
		for _, obj := range bufferForDebug {
			fmt.Printf("%#v\n", obj)
		}
	}()
	var shouldTerminate = false

	go func() {
		for event := range cfg.channel {
			if event == Terminate {
				shouldTerminate = true
			}
			r.eventChan <- event
		}
	}()

	keyChannel := make(chan rune, 1024)
	keyBuffer := make([]rune, 0)
	go func() {
		reader := bufio.NewReaderSize(os.Stdin, 256)
		for {
			ch, _, err := reader.ReadRune()
			if err != nil {
				panic(fmt.Errorf("failed to read keyboard input: %w", err))
			}
			keyChannel <- ch
		}
	}()

	for {
		r.shouldSkipRendering = false
	Depth1:
		for {
			select {
			case k := <-keyChannel:
				keyBuffer = append(keyBuffer, k)
			case <-time.After(time.Millisecond):
				break Depth1
			}
		}
		if len(keyBuffer) == 0 {
			r.shouldSkipRendering = true
		}
		for {
			ch, size := readBuffer(keyBuffer)
			if size == 0 {
				break
			}
			keyBuffer = keyBuffer[size:]
			if ch == key.CtrlC {
				return nil
			}
			if cfg.focusedView != nil {
				switch cfg.focusedView.keyHandler(ch).(type) {
				case terminate:
					return nil
				case nil:
					if cfg.eventHandler != nil {
						switch cfg.eventHandler(ch).(type) {
						case terminate:
							return nil
						}
					}
				}
			} else if cfg.eventHandler != nil {
				switch cfg.eventHandler(ch).(type) {
				case terminate:
					return nil
				}
			}
		}
		event := func() any {
			select {
			case event := <-r.eventChan:
				r.shouldSkipRendering = false
				return event
			case <-time.After(time.Millisecond * 10):
				return nil
			}
		}()

		if cfg.eventHandler != nil {
			switch cfg.eventHandler(event).(type) {
			case terminate:
				return nil
			}
		}

		if changed, _ := r.updateTerminalSize(); changed {
			r.shouldSkipRendering = false
		}

		if shouldTerminate {
			return nil
		}

		if r.shouldSkipRendering {
			continue
		}

		// Clear
		r.fill(style{})

		v := ZStack(createView())
		cfg.focusedView = nil
		// Render views
		err = renderView(r, v, &cfg, rect{0, 0, r.width, r.height}, style{})
		if err != nil {
			return fmt.Errorf("failed to render view: %w", err)
		}

		// Draw
		r.draw()

	}
}

type terminate struct{}

var Terminate = terminate{}

var bufferForDebug = make([]any, 0)

func renderView(r *renderer, v *View, cfg *config, frame rect, defaultStyle style) error {
	vr, err := newViewRenderer(r, frame.x, frame.y, frame.width, frame.height, int(v.paddingTop), int(v.paddingLeading), int(v.paddingBottom), int(v.paddingTrailing))
	if err != nil {
		return fmt.Errorf("failed to create viewRenderer: %w", err)
	}
	if v.style == nil {
		v.style = new(style)
	}
	v.style.merge(defaultStyle)
	if v.border != nil || v.title != "" || v.renderer != nil || v.style.b256 != 0 {
		vr.fill(cell{' ', 1, *v.style})
	}
	if v.border != nil {
		v.border.merge(*v.style)
		vr.putBorder(*v.border)
	}
	if v.title != "" {
		vr.putTitle([]text{{Str: " " + v.title + " ", Style: *v.style}})
	}
	if v.renderer != nil {
		vr.putBody(v.renderer(), *v.style)
	}
	if v.keyHandler != nil && (cfg.focusedView == nil || v.priority >= cfg.focusedView.priority) {
		cfg.focusedView = v
	}

	availableWidth := frame.width - int(v.paddingLeading) - int(v.paddingTrailing)
	availableHeight := frame.height - int(v.paddingTop) - int(v.paddingBottom)

	accumulatedX := frame.x + int(v.paddingLeading)
	accumulatedY := frame.y + int(v.paddingTop)

	remainedWidth := availableWidth
	remainedHeight := availableHeight
	numberOfAutoWidth := 0
	numberOfAutoHeight := 0

	for idx := range v.children {
		if v.children[idx] == nil {
			continue
		}
		if v.children[idx].absoluteWidth == 0 {
			v.children[idx].absoluteWidth = availableWidth * int(v.children[idx].relativeWidth) / 12
		}
		if v.children[idx].absoluteHeight == 0 {
			v.children[idx].absoluteHeight = availableHeight * int(v.children[idx].relativeHeight) / 12
		}

		remainedWidth -= v.children[idx].absoluteWidth
		remainedHeight -= v.children[idx].absoluteHeight

		if v.children[idx].absoluteWidth == 0 {
			numberOfAutoWidth++
		}

		if v.children[idx].absoluteHeight == 0 {
			numberOfAutoHeight++
		}
	}
	for _, child := range v.children {
		if child == nil {
			continue
		}
		if child.absoluteWidth == 0 {
			if v.dir == horizontal {
				child.absoluteWidth = remainedWidth / numberOfAutoWidth
				numberOfAutoWidth--
				remainedWidth -= child.absoluteWidth
			} else {
				child.absoluteWidth = availableWidth
			}
		}
		if child.absoluteHeight == 0 {
			if v.dir == vertical {
				child.absoluteHeight = remainedHeight / numberOfAutoHeight
				numberOfAutoHeight--
				remainedHeight -= child.absoluteHeight
			} else {
				child.absoluteHeight = availableHeight
			}
		}

		x := frame.x + int(v.paddingLeading) + (availableWidth-child.absoluteWidth)/2
		if v.dir == horizontal {
			x = accumulatedX
		}
		y := frame.y + int(v.paddingTop) + (availableHeight-child.absoluteHeight)/2
		if v.dir == vertical {
			y = accumulatedY
		}

		err = renderView(r, child, cfg, rect{
			x,
			y,
			child.absoluteWidth,
			child.absoluteHeight,
		}, *v.style)
		if err != nil {
			return err
		}
		if v.dir == horizontal {
			accumulatedX += child.absoluteWidth
		}
		if v.dir == vertical {
			accumulatedY += child.absoluteHeight
		}
	}
	return nil
}

type rect struct {
	x      int
	y      int
	width  int
	height int
}
