package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Event interface{}

type MouseEvent struct {
	X int32
	Y int32

	ButtonLeft bool
	ButtonMiddle bool
	ButtonRight bool

	OverMe bool
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

	Selectable bool
	Selected bool

	LastRenderedTexture *sdl.Texture
	LastRenderedX int32
	LastRenderedY int32
	LastRenderedWidth int32
	LastRenderedHeight int32
	LastRenderedMarginX int32
	LastRenderedMarginY int32
}

func CreateElement(width int32, widthPercent bool, height int32, heightPercent bool, marginX int32, marginXPercent bool, marginY int32, marginYPercent bool, backgroundColor sdl.Color, scrollX, scrollY bool, selectable bool) *Element {
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

		if e.Selected {
			renderer.SetDrawColor(0, 255, 255, 255)
			drawThickRect(renderer, &sdl.Rect{0, 0, textureWidth, textureHeight}, 2)
		}

		e.LastRenderedWidth = textureWidth
		e.LastRenderedHeight = textureHeight

		return elementTexture, nil
	} else {
		expandedW, expandedH := e.ExpandedSize()

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
				child.LastRenderedMarginX = child.MarginX * expandedW / 100
			}
			if child.MarginYPercent {
				child.LastRenderedMarginY = child.MarginY * expandedH / 100
			}

			maxWidth += width + 2 * child.LastRenderedMarginX

			child.LastRenderedTexture = texture
		}

		if e.Width >= 0 {
			maxWidth = expandedW
		}

		var currentX, currentY int32

		var currentLineHeight int32

		for _, child := range(e.Children) {
			texture := child.LastRenderedTexture

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
			currentY = expandedH
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
			_, _, width, height, err := child.LastRenderedTexture.Query()
			if err != nil {
				return nil, err
			}

			if !e.ScrollX {
				e.ScrollPositionX = 0
			}
			if !e.ScrollY {
				e.ScrollPositionY = 0
			}

			err = renderer.Copy(child.LastRenderedTexture, nil, &sdl.Rect{child.LastRenderedX, child.LastRenderedY, width, height})
			if err != nil {
				return nil, err
			}
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

func (e *Element) MouseUpdate(event MouseEvent) {
	for _, handler := range(e.EventHandlers) {
		handler(event)
	}

	for _, child := range(e.Children) {
		overChild := event.OverMe && event.X >= child.LastRenderedX && event.X < child.LastRenderedX + child.LastRenderedWidth && event.Y >= child.LastRenderedY && event.Y < child.LastRenderedY + child.LastRenderedHeight

		childEvent := MouseEvent{event.X - child.LastRenderedX, event.Y - child.LastRenderedY, event.ButtonLeft, event.ButtonMiddle, event.ButtonRight, overChild}

		child.MouseUpdate(childEvent)
	}

	if event.ButtonLeft {
		e.Selected = e.Selectable && event.OverMe
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

type SDLEventWatch struct{}

func (_ SDLEventWatch) FilterEvent(event sdl.Event, _ any) bool {
	switch e := event.(type) {
	case *sdl.WindowEvent:
		window, err := sdl.GetWindowFromID(e.WindowID)
		if err != nil {
			panic(err)
		}

		switch e.Event {
		case sdl.WINDOWEVENT_RESIZED, sdl.WINDOWEVENT_SIZE_CHANGED:
			if window.GetFlags() & sdl.WINDOW_RESIZABLE != 0 {
				window.SetSize(e.Data1, e.Data2)
			}
		}
	}

	return true
}

func drawThickRect(renderer *sdl.Renderer, rect *sdl.Rect, thickness int32) error {
	err := renderer.FillRect(&sdl.Rect{rect.X, rect.Y, rect.W, thickness})
	if err != nil {
		return err
	}
	err = renderer.FillRect(&sdl.Rect{rect.X, rect.Y, thickness, rect.H})
	if err != nil {
		return err
	}
	err = renderer.FillRect(&sdl.Rect{rect.X + rect.W - thickness, 0, thickness, rect.H})
	if err != nil {
		return err
	}
	err = renderer.FillRect(&sdl.Rect{0, rect.Y + rect.H - thickness, rect.W, thickness})
	if err != nil {
		return err
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

	_, err = window.GetSurface()
	if err != nil {
		panic(err)
	}

	renderer, err := window.GetRenderer()
	if err != nil {
		panic(err)
	}

	sdl.AddEventWatch(SDLEventWatch{}, nil)

	root := CreateElement(1280, false, 720, false, 0, false, 0, false, sdl.Color{64, 64, 64, 255}, false, false, false)

	child := CreateElement(-50, true, 100, true, 0, false, 0, false, sdl.Color{64, 0, 0, 255}, false, false, false)

	root.AppendChild(child)

	childChild := CreateElement(500, false, 50, true, 25, false, 0, false, sdl.Color{0, 255, 0, 255}, false, false, true)

	font, err := ttf.OpenFont("./fonts/Xanh_Mono/XanhMono-Regular.ttf", 24)

	if err != nil {
		panic(err)
	}

	childChild.SetContent(&Text{
		Content: "This is cool",
		Font: font,
		Color: sdl.Color{255, 0, 255, 255},
		Wrap: true,
	})

	child.AppendChild(childChild)

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

		renderer.Present()

		mouseX, mouseY, mouseState := sdl.GetMouseState()

		root.MouseUpdate(MouseEvent{
			mouseX,
			mouseY,
			mouseState & sdl.BUTTON_LEFT != 0,
			mouseState & sdl.BUTTON_MIDDLE != 0,
			mouseState & sdl.BUTTON_RIGHT != 0,
			true,
		})

		sdl.Delay(10)
	}
}
