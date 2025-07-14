package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type ElementContent interface {
	SetContainer(*Element)
	Render(*sdl.Renderer) (*sdl.Texture, error) // should return a texture and not actually render anything on the window
}

type Element struct {
	Width int32
	WidthPercent bool
	Height int32
	HeightPercent bool

	MarginX int32
	MarginY int32

	ScrollX bool
	ScrollPositionX int32
	ScrollY bool
	ScrollPositionY int32

	Parent *Element
	Children []*Element

	Content ElementContent
}

func CreateElement(width int32, widthPercent bool, height int32, heightPercent bool, marginX, marginY int32, scrollX, scrollY bool) Element {
	return Element {
		Width: width,
		WidthPercent: widthPercent,
		Height: height,
		HeightPercent: heightPercent,

		MarginX: marginX,
		MarginY: marginY,

		ScrollX: scrollX,
		ScrollPositionX: 0,
		ScrollY: scrollY,
		ScrollPositionY: 0,

		Parent: nil,
		Children: nil,

		Content: nil,
	}
}

func (e *Element) ExpandedSize() (int32, int32) {
	if e.Width > 0 && e.Height > 0 {
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

	if e.Children == nil {
		if e.Content != nil {
			contentTexture, err := e.Content.Render(renderer)
			if err != nil {
				return nil, err
			}

			_, _, contentWidth, contentHeight, err := contentTexture.Query()
			if err != nil {
				return nil, err
			}

			if realWidth >= 0 {
				contentWidth = realWidth
			}
			if realHeight >= 0 {
				contentHeight = realHeight
			}

			elementTexture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_TARGET, contentWidth, contentHeight)
			if err != nil {
				return nil, err
			}

			err = renderer.SetRenderTarget(elementTexture)
			if err != nil {
				return nil, err
			}

			err = renderer.Copy(contentTexture, &sdl.Rect{0, 0, contentWidth, contentHeight}, nil)
			if err != nil {
				return nil, err
			}

			return elementTexture, nil
		}
	} else {
		e.Content = nil

		expandedW, expandedH := e.ExpandedSize()

		var maxWidth int32

		childTextures := []*sdl.Texture{}

		for _, child := range(e.Children) {
			texture, err := child.Render(renderer)
			if err != nil {
				return nil, err
			}

			_, _, width, _, err := texture.Query()
			if err != nil {
				return nil, err
			}

			maxWidth += width + 2 * child.MarginX

			childTextures = append(childTextures, texture)
		}

		if expandedW >= 0 {
			maxWidth = expandedW
		}

		locations := [][2]int32{}

		var currentX, currentY int32

		var currentLineHeight int32

		for i, child := range(e.Children) {
			texture := childTextures[i]

			_, _, width, height, err := texture.Query()
			if err != nil {
				return nil, err
			}

			totalHeight := height + 2 * child.MarginY

			newX := currentX + width + 2 * child.MarginX

			if newX > maxWidth {
				currentX = 0
				currentY += currentLineHeight
				currentLineHeight = 0
			}

			if totalHeight > currentLineHeight {
				currentLineHeight = totalHeight
			}

			locations = append(locations, [2]int32{currentX + child.MarginX, currentY + child.MarginY})

			currentX = newX
		}

		currentY += currentLineHeight

		if expandedH >= 0 {
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

		for i, texture := range(childTextures) {
			location := locations[i]

			_, _, width, height, err := texture.Query()
			if err != nil {
				return nil, err
			}

			if !e.ScrollX {
				e.ScrollPositionX = 0
			}
			if !e.ScrollY {
				e.ScrollPositionY = 0
			}

			err = renderer.Copy(texture, nil, &sdl.Rect{location[0] - e.ScrollPositionX, location[1] - e.ScrollPositionY, width, height})
			if err != nil {
				return nil, err
			}
		}

		return elementTexture, nil
	}

	if realWidth < 0 {
		realWidth = 0
	}

	if realHeight < 0 {
		realHeight = 0
	}

	elementTexture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_TARGET, realWidth, realHeight)
	if err != nil {
		return nil, err
	}

	return elementTexture, nil
}

func (e *Element) AppendChild(child *Element) {
	if e.Children == nil {
		e.Children = []*Element{}
	}

	e.Content = nil

	e.Children = append(e.Children, child)
}

func (e *Element) AddContent(content ElementContent) {
	if content == nil {
		return
	}

	e.Children = nil
	e.Content = content
	content.SetContainer(e)
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


	root := CreateElement(1280, false, 720, false, 0, 0, false, false)

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
		renderer.Clear()

		texture, err := root.Render(renderer)
		if err != nil {
			panic(err)
		}

		renderer.Copy(texture, &sdl.Rect{0, 0, windowW, windowH}, &sdl.Rect{0, 0, windowW, windowH})

		renderer.Present()

		sdl.Delay(10)
	}
}
