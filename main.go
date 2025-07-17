package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Event interface{}

type MouseHoverEvent bool

const (
	MouseButtonEventDown = iota
	MouseButtonEventUp
	MouseButtonEventClick
)

type MouseButtonEvent struct {
	Type int
	Button int
}

type ElementContent interface {
	SetContainer(*Element)
	Render(*sdl.Renderer) (*sdl.Texture, error)
}

type Text struct {
	Container *Element

	Content string

	Font *ttf.Font
	Color sdl.Color
	Wrap bool
}

func (t *Text) SetContainer(e *Element) {
	t.Container = e
}

func (t *Text) Render(renderer *sdl.Renderer) (*sdl.Texture, error) {
	var surface *sdl.Surface

	parentW, _ := t.Container.ExpandedSize()

	var err error

	if t.Wrap && parentW >= 0 {
		surface, err = t.Font.RenderUTF8BlendedWrapped(t.Content, t.Color, int(parentW))
	} else {
		surface, err = t.Font.RenderUTF8Blended(t.Content, t.Color)
	}

	if err != nil {
		return nil, err
	}

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return nil, err
	}

	surface.Free()

	return texture, nil
}

type Element struct {
	Width int32
	WidthPercent bool
	Height int32
	HeightPercent bool

	MarginX int32
	MarginXPercent bool
	MarginY int32
	MarginYPercent bool

	BackgroundColor sdl.Color

	ScrollX bool
	ScrollPositionX int32
	ScrollY bool
	ScrollPositionY int32

	Parent *Element
	Children []*Element

	Content ElementContent

	EventHandlers []func(Event)

	IsTextInput bool

	Selectable bool
	Selected bool

	MouseHovering bool
	MouseButtonClicks [3]bool

	RenderingTexture *sdl.Texture

	LastRenderedX int32
	LastRenderedY int32
	LastRenderedWidth int32
	LastRenderedHeight int32
	LastRenderedMarginX int32
	LastRenderedMarginY int32
}

func CreateElement(width int32, widthPercent bool, height int32, heightPercent bool, marginX int32, marginXPercent bool, marginY int32, marginYPercent bool, backgroundColor sdl.Color, scrollX, scrollY, isTextInput, selectable bool) *Element {
	return &Element {
		Width: width,
		WidthPercent: widthPercent,
		Height: height,
		HeightPercent: heightPercent,

		MarginX: marginX,
		MarginXPercent: marginXPercent,
		MarginY: marginY,
		MarginYPercent: marginYPercent,

		BackgroundColor: backgroundColor,

		ScrollX: scrollX,
		ScrollY: scrollY,

		EventHandlers: []func(Event){},

		IsTextInput: isTextInput,

		Selectable: selectable,
	}
}

func (e *Element) ExpandedSize() (int32, int32) {
	if !e.WidthPercent && !e.HeightPercent {
		return e.Width, e.Height
	}

	realWidth, realHeight := e.Width, e.Height

	var parentW, parentH int32 = 0, 0

	if e.Parent != nil {
		parentW, parentH = e.Parent.ExpandedSize()
	}

	if e.WidthPercent {
		realWidth = e.Width * parentW / 100
	}

	if e.HeightPercent {
		realHeight = e.Height * parentH / 100
	}

	return realWidth, realHeight
}

func (e *Element) Render(renderer *sdl.Renderer) (*sdl.Texture, error) {
	realWidth, realHeight := e.ExpandedSize()

	if e.Content != nil {
		e.Children = nil

		contentTexture, err := e.Content.Render(renderer)
		if err != nil {
			return nil, err
		}

		_, _, contentWidth, contentHeight, err := contentTexture.Query()
		if err != nil {
			return nil, err
		}

		textureWidth, textureHeight := contentWidth, contentHeight

		if realWidth >= 0 {
			textureWidth = realWidth
		}
		if realHeight >= 0 {
			textureHeight = realHeight
		}

		elementTexture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_TARGET, textureWidth, textureHeight)
		if err != nil {
			return nil, err
		}

		err = renderer.SetRenderTarget(elementTexture)
		if err != nil {
			return nil, err
		}

		renderer.SetDrawColor(e.BackgroundColor.R, e.BackgroundColor.G, e.BackgroundColor.B, e.BackgroundColor.A)
		err = renderer.Clear()
		if err != nil {
			return nil, err
		}

		err = renderer.Copy(contentTexture, &sdl.Rect{0, 0, contentWidth, contentHeight}, &sdl.Rect{0, 0, contentWidth, contentHeight})
		if err != nil {
			return nil, err
		}

		contentTexture.Destroy()

		if e.Selected {
			renderer.SetDrawColor(0, 255, 255, 255)
			drawThickRect(renderer, &sdl.Rect{0, 0, textureWidth, textureHeight}, 2)
		}

		e.LastRenderedWidth = textureWidth
		e.LastRenderedHeight = textureHeight

		return elementTexture, nil
	} else {
		var maxWidth int32

		for _, child := range(e.Children) {
			texture, err := child.Render(renderer)
			if err != nil {
				return nil, err
			}

			_, _, width, _, err := texture.Query()
			if err != nil {
				return nil, err
			}

			child.LastRenderedMarginX = child.MarginX
			child.LastRenderedMarginY = child.MarginY

			if child.MarginXPercent {
				child.LastRenderedMarginX = child.MarginX * realWidth / 100
			}
			if child.MarginYPercent {
				child.LastRenderedMarginY = child.MarginY * realHeight / 100
			}

			maxWidth += width + 2 * child.LastRenderedMarginX

			child.RenderingTexture = texture
		}

		if e.Width >= 0 {
			maxWidth = realWidth
		}

		var currentX, currentY int32

		var currentLineHeight int32

		for _, child := range(e.Children) {
			texture := child.RenderingTexture

			_, _, width, height, err := texture.Query()
			if err != nil {
				return nil, err
			}

			totalHeight := height + 2 * child.LastRenderedMarginY

			newX := currentX + width + 2 * child.LastRenderedMarginX

			if newX > maxWidth {
				currentX = 0
				currentY += currentLineHeight
				currentLineHeight = 0
			}

			if totalHeight > currentLineHeight {
				currentLineHeight = totalHeight
			}

			child.LastRenderedX = currentX + child.LastRenderedMarginX - e.ScrollPositionX
			child.LastRenderedY = currentY + child.LastRenderedMarginY - e.ScrollPositionY

			currentX = newX
		}

		currentY += currentLineHeight

		if e.Height >= 0 {
			currentY = realHeight
		}

		elementTexture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_TARGET, maxWidth, currentY)
		if err != nil {
			return nil, err
		}

		err = renderer.SetRenderTarget(elementTexture)
		if err != nil {
			return nil, err
		}

		renderer.SetDrawColor(e.BackgroundColor.R, e.BackgroundColor.G, e.BackgroundColor.B, e.BackgroundColor.A)
		err = renderer.Clear()
		if err != nil {
			return nil, err
		}

		for _, child := range(e.Children) {
			_, _, width, height, err := child.RenderingTexture.Query()
			if err != nil {
				return nil, err
			}

			if !e.ScrollX {
				e.ScrollPositionX = 0
			}
			if !e.ScrollY {
				e.ScrollPositionY = 0
			}

			err = renderer.Copy(child.RenderingTexture, nil, &sdl.Rect{child.LastRenderedX, child.LastRenderedY, width, height})
			if err != nil {
				return nil, err
			}

			child.RenderingTexture.Destroy()
		}

		e.LastRenderedWidth = maxWidth
		e.LastRenderedHeight = currentY

		return elementTexture, nil
	}

	return nil, nil
}

func (e *Element) AddEventHandler(handler func(Event)) {
	e.EventHandlers = append(e.EventHandlers, handler)
}

func (e *Element) Deselect() {
	if e.Selectable {
		e.Selected = false
	}

	for _, child := range(e.Children) {
		child.Deselect()
	}
}

func (e *Element) Emit(event Event) {
	for _, handler := range(e.EventHandlers) {
		handler(event)
	}
}

func (e *Element) MouseUpdate(root *Element, x, y int32, oldMouseButtonStates [3]bool, newMouseButtonStates [3]bool, overMe bool) {
	if overMe && !e.MouseHovering {
		e.MouseHovering = true

		e.Emit(MouseHoverEvent(true))
	}

	if !overMe && e.MouseHovering {
		e.MouseHovering = false

		e.Emit(MouseHoverEvent(false))
	}

	for b, newState := range(newMouseButtonStates) {
		oldState := oldMouseButtonStates[b]

		if !oldState && newState && overMe {
			e.MouseButtonClicks[b] = true

			root.Deselect()

			e.Emit(MouseButtonEvent{MouseButtonEventDown, b})

			if e.Selectable {
				e.Selected = true
			}
		}

		if !overMe {
			e.MouseButtonClicks[b] = false
		}

		if oldState && !newState && overMe {
			e.Emit(MouseButtonEvent{MouseButtonEventUp, b})

			if e.MouseButtonClicks[b] {
				e.Emit(MouseButtonEvent{MouseButtonEventClick, b})
			}
		}
	}

	for _, child := range(e.Children) {
		overChild := overMe && x >= child.LastRenderedX && x < child.LastRenderedX + child.LastRenderedWidth && y >= child.LastRenderedY && y < child.LastRenderedY + child.LastRenderedHeight

		child.MouseUpdate(root, x - child.LastRenderedX, y - child.LastRenderedY, oldMouseButtonStates, newMouseButtonStates, overChild)
	}
}

func (e *Element) AppendChild(child *Element) {
	e.Content = nil

	child.Parent = e

	e.Children = append(e.Children, child)
}

func (e *Element) SetContent(content ElementContent) {
	e.Children = nil

	e.Content = content

	if content != nil {
		content.SetContainer(e)
	}
}

func (e *Element) Locate() (int32, int32) {
	if e.Parent != nil {
		parentX, parentY := e.Parent.Locate()

		return e.LastRenderedX + parentX, e.LastRenderedY + parentY
	}

	return 0, 0
}

func drawThickRect(renderer *sdl.Renderer, rect *sdl.Rect, thickness int32) error {
	thickness--
	for ; thickness >= 0; thickness-- {
		err := renderer.DrawRect(&sdl.Rect{rect.X + thickness, rect.Y + thickness, rect.W - 2 * thickness, rect.H - 2 * thickness})
		if err != nil {
			return err
		}
	}

	return nil
}

func checkForTextInput(e *Element) *Element {
	if e.Selected && e.IsTextInput {
		return e
	}

	for _, child := range(e.Children) {
		textInputElement := checkForTextInput(child)

		if textInputElement != nil {
			return textInputElement
		}
	}

	return nil
}

func main() {
	sdl.Init(sdl.INIT_EVERYTHING)

	ttf.Init()

	window, err := sdl.CreateWindow("AshKmodify", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1280, 720, sdl.WINDOW_OPENGL | sdl.WINDOW_RESIZABLE | sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_SOFTWARE)
	if err != nil {
		panic(err)
	}

	root := CreateElement(1280, false, 720, false, 0, false, 0, false, sdl.Color{0x29, 0x2c, 0x30, 255}, false, false, false, false)

	var oldMouseButtonStates [3]bool

	running := true
	for running {
		for {
			event := sdl.PollEvent()
			if event == nil {
				break
			}

			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
				break
			}
		}

		windowW, windowH := window.GetSize()
		root.Width = windowW
		root.Height = windowH

		renderer.SetDrawColor(0, 0, 0, 255)
		err = renderer.Clear()
		if err != nil {
			panic(err)
		}

		texture, err := root.Render(renderer)
		if err != nil {
			panic(err)
		}

		err = renderer.SetRenderTarget(nil)
		if err != nil {
			panic(err)
		}

		renderer.Copy(texture, &sdl.Rect{0, 0, windowW, windowH}, &sdl.Rect{0, 0, windowW, windowH})

		texture.Destroy()

		renderer.Present()

		mouseX, mouseY, mouseState := sdl.GetMouseState()

		buttonLeft, buttonMiddle, buttonRight := mouseState & 1 != 0, mouseState & 2 != 0, mouseState & 4 != 0

		newMouseButtonStates := [3]bool{buttonLeft, buttonMiddle, buttonRight}

		root.MouseUpdate(root, mouseX, mouseY, oldMouseButtonStates, newMouseButtonStates, true)

		oldMouseButtonStates = newMouseButtonStates

		textInputElement := checkForTextInput(root)

		if textInputElement != nil {
			inputElementX, inputElementY := textInputElement.Locate()

			sdl.SetTextInputRect(&sdl.Rect{inputElementX, inputElementY, textInputElement.LastRenderedWidth, textInputElement.LastRenderedHeight})

			if !sdl.IsTextInputActive() {
				sdl.StartTextInput()
			}
		} else if sdl.IsTextInputActive() {
			sdl.StopTextInput()
		}

		sdl.Delay(20)
	}
}
