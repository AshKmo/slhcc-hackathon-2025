package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Element interface {
	GetPosition() (int32, int32)
	SetPosition(int32, int32)
	Size() (int32, int32)
	Render(*sdl.Renderer, int32, int32) error
}

type TextBox struct {
	Editable bool
	Font *ttf.Font
	Color sdl.Color
	WrapLength int

	x int32
	y int32
	content string
	renderWidth int32
	renderHeight int32
}

func NewTextBox(x, y int32, editable bool, font *ttf.Font, color sdl.Color, wrapLength int, content string) TextBox {
	return TextBox{editable, font, color, wrapLength, x, y, content, 0, 0}
}

func (t TextBox) GetPosition() (int32, int32) {
	return t.x, t.y
}

func (t *TextBox) SetPosition(x, y int32) {
	t.x = x
	t.y = y
}

func (t *TextBox) SetContent() {
	t.renderWidth = 0
	t.renderHeight = 0
	t.Size()
}

func (t TextBox) renderToSurface() (*sdl.Surface, error) {
	if t.WrapLength == 0 {
		return t.Font.RenderUTF8Blended(t.content, t.Color)
	} else {
		return t.Font.RenderUTF8BlendedWrapped(t.content, t.Color, t.WrapLength)
	}
}

func (t *TextBox) Size() (int32, int32) {
	if t.renderWidth == 0 || t.renderHeight == 0 {
		surface, err := t.renderToSurface()
		if err != nil {
			panic(err)
		}

		t.renderWidth = surface.W
		t.renderHeight = surface.H
	}

	return t.renderWidth, t.renderHeight
}

func (t TextBox) Render(renderer *sdl.Renderer, x int32, y int32) error {
	surface, err := t.renderToSurface()
	if err != nil {
		return err
	}

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return err
	}

	sizeX, sizeY := t.Size()

	return renderer.Copy(texture, nil, &sdl.Rect{x + t.x, y + t.y, sizeX, sizeY})
}

type SDLEventWatch struct {}

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

	font, err := ttf.OpenFont("./fonts/Xanh_Mono/XanhMono-Regular.ttf", 24)
	if err != nil {
		panic(err)
	}

	textbox := NewTextBox(0, 0, false, font, sdl.Color{0, 255, 0, 255}, 0, "This is awesome!")

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

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()

		err := textbox.Render(renderer, 0, 0)
		if err != nil {
			panic(err)
		}

		renderer.Present()

		sdl.Delay(10)
	}
}
