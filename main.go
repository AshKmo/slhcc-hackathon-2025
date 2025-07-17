package main

import (
	"errors"
	"os"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"github.com/veandco/go-sdl2/img"
)

type Cursor struct {
	CursorElement *Element

	X int
	Y int
}

func NewCursor() *Cursor {
	return &Cursor{
		&Element{
			Width: 5,
			Height: 40,
			HeightPercent: false,

			BackgroundColor: sdl.Color{0, 255, 255, 0},
		},

		0,
		0,
	}
}

type Event interface{}

type TextEvent byte

type KeyEvent struct {
	Type uint32
	Code sdl.Keycode
}

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
	Destroy() error
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

func (t *Text) Destroy() error {
	return nil
}

type Image struct {
	Container *Element

	ImageSurface *sdl.Surface
}

func (image *Image) SetContainer(e *Element) {
	image.Container = e;
}

func (image *Image) Render(renderer *sdl.Renderer) (*sdl.Texture, error) {
	texture, err := renderer.CreateTextureFromSurface(image.ImageSurface)
	if err != nil {
		return nil, err
	}

	return texture, nil
}

func (image *Image) Destroy() error {
	image.ImageSurface.Free()

	return nil
}

type Element struct {
	Width int32
	MinWidth int32
	WidthPercent bool
	Height int32
	MinHeight int32
	HeightPercent bool

	MarginX int32
	MarginXPercent bool
	MarginY int32
	MarginYPercent bool

	Breaking bool
	Inline bool

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
	LastRenderedChildWidth int32
	LastRenderedChildHeight int32
}

func (e *Element) Destroy() error {
	e.Remove()

	if e.Content != nil {
		err := e.Content.Destroy()
		if err != nil {
			return err
		}
	}

	for _, child := range(e.Children) {
		err := child.Destroy()
		if err != nil {
			return err
		}
	}

	return nil
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

func drawSelectionBorder(renderer *sdl.Renderer, dimensions *sdl.Rect) {
	renderer.SetDrawColor(0, 255, 255, 255)
	drawThickRect(renderer, dimensions, 1)
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

		if textureWidth < e.MinWidth {
			textureWidth = e.MinWidth
		}
		if textureHeight < e.MinHeight {
			textureWidth = e.MinHeight
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
			drawSelectionBorder(renderer, &sdl.Rect{0, 0, textureWidth, textureHeight})
		}

		e.LastRenderedWidth = textureWidth
		e.LastRenderedHeight = textureHeight

		return elementTexture, nil
	} else {
		maxWidth := realWidth

		var currentX, currentY int32

		var currentLineHeight int32

		var childWidth int32

		for i, child := range(e.Children) {
			texture, err := child.Render(renderer)
			if err != nil {
				return nil, err
			}

			child.RenderingTexture = texture

			_, _, width, height, err := texture.Query()
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

			totalHeight := height + child.LastRenderedMarginY

			newX := currentX + width + child.LastRenderedMarginX

			if !child.Inline && (newX > maxWidth && !e.ScrollX) || (i > 0 && e.Children[i - 1].Breaking) {
				if currentX > childWidth {
					childWidth = currentX
				}

				currentX = 0
				newX = currentX + width

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

		if currentX > childWidth {
			childWidth = currentX
		}

		e.LastRenderedChildWidth = childWidth;

		currentY += currentLineHeight

		e.LastRenderedChildHeight = currentY

		if e.Height >= 0 {
			currentY = realHeight
		}

		if e.Width >= 0 {
			maxWidth = realWidth
		}

		if maxWidth < e.MinWidth {
			maxWidth = e.MinWidth
		}

		if currentY < e.MinHeight {
			currentY = e.MinHeight
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

		if e.Selected {
			drawSelectionBorder(renderer, &sdl.Rect{0, 0, maxWidth, currentY})
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

func (e *Element) Emit(event Event) {
	for _, handler := range(e.EventHandlers) {
		handler(event)
	}
}

func inBox(x, y, boxX, boxY, boxW, boxH int32) bool {
	return x >= boxX && x < boxX + boxW && y >= boxY && y < boxY + boxH
}

func (e *Element) MouseUpdate(root *Element, selected **Element, x, y int32, oldMouseButtonStates [3]bool, newMouseButtonStates [3]bool, overMe bool) {
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

			e.Emit(MouseButtonEvent{MouseButtonEventDown, b})

			if e.Selectable {
				*selected = e
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

		if !newState {
			e.MouseButtonClicks[b] = false
		}
	}

	for _, child := range(e.Children) {
		overChild := overMe && inBox(x, y, child.LastRenderedX, child.LastRenderedY, child.LastRenderedWidth, child.LastRenderedHeight)

		child.MouseUpdate(root, selected, x - child.LastRenderedX, y - child.LastRenderedY, oldMouseButtonStates, newMouseButtonStates, overChild)
	}
}

func (e *Element) Scroll(mouseX, mouseY, scrollX, scrollY int32) bool {
	if e.Content != nil {
		return false;
	}

	for _, child := range(e.Children) {
		if child.Scroll(mouseX, mouseY, scrollX, scrollY) {
			return true;
		}
	}

	if !inBox(mouseX, mouseY, e.LastRenderedX, e.LastRenderedY, e.LastRenderedWidth, e.LastRenderedHeight) {
		return false;
	}

	scrolled := false

	if !e.ScrollY || e.LastRenderedHeight >= e.LastRenderedChildHeight {
		scrollX = scrollY
	}

	for i := 0; i < 3; i++ {
		if e.ScrollX {
			if e.ScrollPositionX + e.LastRenderedWidth >= e.LastRenderedChildWidth {
				if e.LastRenderedChildWidth >= e.LastRenderedWidth {
					e.ScrollPositionX = e.LastRenderedChildWidth - e.LastRenderedWidth;
				} else {
					e.ScrollPositionX = 0
				}
			} else if e.ScrollPositionX <= 0 {
				e.ScrollPositionX = 0
			} else if i == 0 {
				scrolled = true

				e.ScrollPositionX -= scrollX * 60
			}
		}

		if e.ScrollY {
			if e.ScrollPositionY + e.LastRenderedHeight > e.LastRenderedChildHeight {
				if e.LastRenderedChildHeight >= e.LastRenderedHeight {
					e.ScrollPositionY = e.LastRenderedChildHeight - e.LastRenderedHeight;
				} else {
					e.ScrollPositionY = 0
				}
			} else if e.ScrollPositionY < 0 {
				e.ScrollPositionY = 0
			} else if i == 0 {
				scrolled = true

				e.ScrollPositionY -= scrollY * 60
			}
		}
	}

	return scrolled
}

func (e *Element) AppendChild(child *Element) {
	e.Content = nil

	child.Parent = e

	e.Children = append(e.Children, child)
}

func (e *Element) InsertChild(child *Element, i int) {
	if i >= len(e.Children) {
		e.AppendChild(child)
		return
	}

	e.Content = nil

	child.Parent = e

	e.Children = append(e.Children[:i], append([]*Element{child}, e.Children[i:]...)...)
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

func (e *Element) DeselectAll() {
	e.Selected = false

	for _, child := range(e.Children) {
		child.DeselectAll()
	}
}

func (e *Element) Remove() {
	if e.Parent == nil {
		return
	}

	i := 0

	for ; i < len(e.Parent.Children); i++ {
		if e.Parent.Children[i] == e {
			break
		}
	}

	if i < len(e.Parent.Children) {
		e.Parent.Children = append(e.Parent.Children[:i], e.Parent.Children[i + 1:]...)
		e.Parent = nil
	}
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

func loadImage(file string) (*Image, error) {
	image := &Image{}

	surface, err := img.Load(file)
	if err != nil {
		return nil, err
	}

	image.ImageSurface = surface

	return image, nil
}

func newCharElement(font *ttf.Font, c byte, textEditingArea *Element, cursor *Cursor) *Element {
	charElement := &Element{
		Width: -1,
		Height: -1,
	}

	charElement.AddEventHandler(func(event Event) {
		switch e := event.(type) {
		case MouseButtonEvent:
			if e.Type == MouseButtonEventDown && e.Button == 0 {
				for y, row := range(textEditingArea.Children) {
					for x, char := range(row.Children) {
						if char == charElement {
							cursor.CursorElement.Remove()
							cursor.X = x + 1
							cursor.Y = y
							row.InsertChild(cursor.CursorElement, cursor.X)
							break
						}
					}
				}
			}
		}
	})

	text := &Text{
		Content: string(c),
		Font: font,
		Color: sdl.Color{255, 255, 255, 255},
	}

	charElement.SetContent(text)

	return charElement
}

func newLineElement(textEditingArea *Element, cursor *Cursor) *Element {
	lineElement := &Element{
		Width: 100,
		WidthPercent: true,
		Height: -1,
		MinHeight: 40,
	}

	lineElement.AddEventHandler(func(event Event) {
		switch e := event.(type) {
		case MouseButtonEvent:
			if e.Type == MouseButtonEventDown && e.Button == 0 {
				for y, row := range(textEditingArea.Children) {
					if row == lineElement {
						cursor.CursorElement.Remove()
						cursor.X = len(lineElement.Children)
						cursor.Y = y
						row.InsertChild(cursor.CursorElement, cursor.X)
						break
					}
				}
			}
		}
	})

	return lineElement
}

func writeChar(textEditingArea *Element, font *ttf.Font, cursor *Cursor, c byte) {
	switch c {
	case '\b':
		if cursor.X == 0 {
			if cursor.Y != 0 {
				cursor.CursorElement.Remove()

				lastRow := textEditingArea.Children[cursor.Y]

				cursor.Y--

				removed := 0

				for len(lastRow.Children) > 0 {
					child := lastRow.Children[0]
					child.Remove()
					textEditingArea.Children[cursor.Y].AppendChild(child)
					removed++
				}

				lastRow.Remove()
				cursor.X = len(textEditingArea.Children[cursor.Y].Children) - removed
				textEditingArea.Children[cursor.Y].InsertChild(cursor.CursorElement, cursor.X)
			}
		} else {
			textEditingArea.Children[cursor.Y].Children[cursor.X - 1].Remove()
			cursor.X--
		}
	case '\x7F':
		if cursor.X + 1 < len(textEditingArea.Children[cursor.Y].Children) {
			textEditingArea.Children[cursor.Y].Children[cursor.X + 1].Remove()
		} else if cursor.Y + 1 < len(textEditingArea.Children) {
			nextRow := textEditingArea.Children[cursor.Y + 1]

			for len(nextRow.Children) > 0 {
				child := nextRow.Children[0]
				child.Remove()
				textEditingArea.Children[cursor.Y].AppendChild(child)
			}

			nextRow.Remove()
		}
	case '\n':
		line := newLineElement(textEditingArea, cursor)
		textEditingArea.InsertChild(line, cursor.Y + 1)

		for len(textEditingArea.Children[cursor.Y].Children) > cursor.X {
			child := textEditingArea.Children[cursor.Y].Children[cursor.X]
			child.Remove()
			line.AppendChild(child)
		}

		cursor.CursorElement.Remove()
		line.InsertChild(cursor.CursorElement, 0)
		cursor.Y++
		cursor.X = 0
	default:
		textEditingArea.Children[cursor.Y].InsertChild(newCharElement(font, c, textEditingArea, cursor), cursor.X)
		cursor.X++
	}
}

func main() {
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}

	ttf.Init()
	if err != nil {
		panic(err)
	}

	img.Init(img.INIT_PNG)
	if err != nil {
		panic(err)
	}

	if len(os.Args) != 2 {
		sdl.ShowMessageBox(&sdl.MessageBoxData{
			Flags: sdl.MESSAGEBOX_ERROR,
			Title: "AshKmodify Error",
			Message: "AshKmodify must be executed with exactly one command line argument specifying the file that it is to edit",
			Buttons: []sdl.MessageBoxButtonData{
				{
					Flags: sdl.MESSAGEBOX_BUTTON_RETURNKEY_DEFAULT,
					Text: "Bother",
				},
			},
		})

		return
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	window, err := sdl.CreateWindow("AshKmodify: " + filePath, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1280, 720, sdl.WINDOW_OPENGL | sdl.WINDOW_RESIZABLE | sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}

	font, err := ttf.OpenFont("./fonts/Xanh_Mono/XanhMono-Regular.ttf", 30)
	if err != nil {
		panic(err)
	}

	var selectedElement *Element

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_SOFTWARE)
	if err != nil {
		panic(err)
	}

	root := &Element{
		Width: 1280,
		Height: 720,
	}

	topBar := &Element{
		Width: 100,
		WidthPercent: true,
		Height: 40,

		BackgroundColor: sdl.Color{32, 32, 32, 255},
	}
	root.AppendChild(topBar)

	/*buttonNew := &Element{
		Width: 30,
		Height: 30,

		MarginX: 5,
		MarginY: 5,
	}

	imgButtonNew, err := loadImage("images/new.png")
	if err != nil {
		panic(err)
	}

	buttonNew.SetContent(imgButtonNew)

	topBar.AppendChild(buttonNew)

	buttonLoad := &Element{
		Width: 30,
		Height: 30,

		MarginX: 5,
		MarginY: 5,
	}

	imgButtonLoad, err := loadImage("images/load.png")
	if err != nil {
		panic(err)
	}

	buttonLoad.SetContent(imgButtonLoad)

	topBar.AppendChild(buttonLoad)*/

	buttonSave := &Element{
		Width: 30,
		Height: 30,

		MarginX: 5,
		MarginY: 5,
	}

	imgButtonSave, err := loadImage("images/save.png")
	if err != nil {
		panic(err)
	}

	buttonSave.SetContent(imgButtonSave)

	topBar.AppendChild(buttonSave)

	textEditingArea := &Element{
		Width: 100,
		WidthPercent: true,
		Height: 100,

		ScrollY: true,

		Selectable: true,
		Selected: true,

		IsTextInput: true,

		BackgroundColor: sdl.Color{64, 64, 64, 255},
	}

	cursor := NewCursor()

	firstLine := newLineElement(textEditingArea, cursor)

	textEditingArea.AppendChild(firstLine)

	firstLine.AppendChild(cursor.CursorElement)

	textEditingArea.AddEventHandler(func(event Event) {
		switch e := event.(type) {
		case TextEvent:
			writeChar(textEditingArea, font, cursor, byte(e))
		case KeyEvent:
			if e.Type == sdl.KEYDOWN {
				switch e.Code {
				case sdl.K_RETURN:
					writeChar(textEditingArea, font, cursor, '\n')
				case sdl.K_BACKSPACE:
					writeChar(textEditingArea, font, cursor, '\b')
				case sdl.K_DELETE:
					writeChar(textEditingArea, font, cursor, '\x7F')
				case sdl.K_RIGHT, sdl.K_LEFT, sdl.K_DOWN, sdl.K_UP:
					cursor.CursorElement.Remove()

					switch e.Code {
					case sdl.K_RIGHT:
						if cursor.X < len(textEditingArea.Children[cursor.Y].Children) {
							cursor.X++
						} else if cursor.Y + 1 < len(textEditingArea.Children) {
							cursor.Y++
							cursor.X = 0
						}
					case sdl.K_LEFT:
						if cursor.X > 0 {
							cursor.X--
						} else if cursor.Y > 0 {
							cursor.Y--
							cursor.X = len(textEditingArea.Children[cursor.Y].Children)
						}
					case sdl.K_UP:
						if cursor.Y > 0 {
							cursor.Y--
							cursor.X = min(cursor.X, len(textEditingArea.Children[cursor.Y].Children))
						}
					case sdl.K_DOWN:
						if cursor.Y + 1 < len(textEditingArea.Children) {
							cursor.Y++
							cursor.X = min(cursor.X, len(textEditingArea.Children[cursor.Y].Children))
						}
					}

					textEditingArea.Children[cursor.Y].InsertChild(cursor.CursorElement, cursor.X)
				}
			}
		}
	})

	root.AppendChild(textEditingArea)

	buttonSave.AddEventHandler(func(event Event) {
		switch e := event.(type) {
		case MouseButtonEvent:
			if e.Type == MouseButtonEventClick && e.Button == 0 {
				file, err := os.Create(filePath)
				defer file.Close()

				if err != nil {
					sdl.ShowMessageBox(&sdl.MessageBoxData{
						Flags: sdl.MESSAGEBOX_ERROR,
						Title: "AshKmodify Error",
						Message: "The file cannot be opened for writing",
						Buttons: []sdl.MessageBoxButtonData{
							{
								Flags: sdl.MESSAGEBOX_BUTTON_RETURNKEY_DEFAULT,
								Text: "Bother",
							},
						},
					})

					return
				}

				for i, row := range(textEditingArea.Children) {
					for _, c := range(row.Children) {
						if c.Content == nil {
							continue
						}

						_, err := file.WriteString(c.Content.(*Text).Content)
						if err != nil {
							goto bother
						}
					}

					if i + 1 < len(textEditingArea.Children) {
						_, err := file.WriteString("\n")
						if err != nil {
							goto bother
						}
					}
				}

				sdl.ShowMessageBox(&sdl.MessageBoxData{
					Flags: sdl.MESSAGEBOX_INFORMATION,
					Title: "AshKmodify",
					Message: "File saved successfully",
					Buttons: []sdl.MessageBoxButtonData{
						{
							Flags: sdl.MESSAGEBOX_BUTTON_RETURNKEY_DEFAULT,
							Text: "Fantastic",
						},
					},
				})

				return

				bother:
				sdl.ShowMessageBox(&sdl.MessageBoxData{
					Flags: sdl.MESSAGEBOX_ERROR,
					Title: "AshKmodify Error",
					Message: "There was an error while writing to the file",
					Buttons: []sdl.MessageBoxButtonData{
						{
							Flags: sdl.MESSAGEBOX_BUTTON_RETURNKEY_DEFAULT,
							Text: "Bother",
						},
					},
				})

			}
		}
	})

	selectedElement = textEditingArea

	for _, c := range(data) {
		writeChar(textEditingArea, font, cursor, c)
	}

	var oldMouseButtonStates [3]bool

	running := true
	for running {
		if selectedElement != nil && selectedElement.IsTextInput {
			inputElementX, inputElementY := selectedElement.Locate()

			sdl.SetTextInputRect(&sdl.Rect{inputElementX, inputElementY, selectedElement.LastRenderedWidth, selectedElement.LastRenderedHeight})

			if !sdl.IsTextInputActive() {
				sdl.StartTextInput()
			}
		} else if sdl.IsTextInputActive() {
			sdl.StopTextInput()
		}

		for {
			event := sdl.PollEvent()
			if event == nil {
				break
			}

			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
				break
			case *sdl.MouseMotionEvent, *sdl.MouseButtonEvent:
				mouseX, mouseY, mouseState := sdl.GetMouseState()
				buttonLeft, buttonMiddle, buttonRight := mouseState & 1 != 0, mouseState & 2 != 0, mouseState & 4 != 0
				newMouseButtonStates := [3]bool{buttonLeft, buttonMiddle, buttonRight}
				root.MouseUpdate(root, &selectedElement, mouseX, mouseY, oldMouseButtonStates, newMouseButtonStates, true)
				oldMouseButtonStates = newMouseButtonStates

				root.DeselectAll()

				if selectedElement != nil && selectedElement.Selectable {
					selectedElement.Selected = true
				}
			case *sdl.MouseWheelEvent:
				mouseX, mouseY, _ := sdl.GetMouseState()
				scrollX, scrollY := e.X, e.Y
				root.Scroll(mouseX, mouseY, scrollX, scrollY)
			case *sdl.TextInputEvent:
				if sdl.IsTextInputActive() {
					selectedElement.Emit(TextEvent(e.Text[0]))
				}
			case *sdl.KeyboardEvent:
				if selectedElement != nil {
					selectedElement.Emit(KeyEvent{e.Type, e.Keysym.Sym})
				}
			}
		}

		windowW, windowH := window.GetSize()

		root.Width = windowW
		root.Height = windowH

		textEditingArea.Height = windowH - topBar.Height

		err := renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
		if err != nil {
			panic(err)
		}

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

		sdl.Delay(20)
	}
}
