package renderers

import (
	"fmt"
	"unsafe"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

type SDLRenderer struct {
	fontTexture *sdl.Texture
	sdlRenderer *sdl.Renderer
}

func NewSDLRenderer(sdlRenderer *sdl.Renderer) (*SDLRenderer, error) {
	renderer:= &SDLRenderer {
		sdlRenderer: sdlRenderer,
	}
	renderer.createDeviceObjects()
	return renderer, nil
}

func (renderer *SDLRenderer) Dispose() {
	renderer.destroyDeviceObjects()
}

func (renderer *SDLRenderer) destroyFontsTexture() {
	if renderer.fontTexture != nil {
		imgui.CurrentIO().Fonts().SetTextureID(0)
		renderer.fontTexture.Destroy()
		renderer.fontTexture = nil
	}
}

func (renderer *SDLRenderer) createFontsTexture() {
	var err error
	io := imgui.CurrentIO()
	image := io.Fonts().TextureDataRGBA32()

	// Load as RGBA 32-bit (75% of the memory is wasted, but default font is so small) because it is more likely to be compatible with user's existing shaders. If your ImTextureId represent a higher-level concept than just a GL texture id, consider calling GetTexDataAsAlpha8() instead to save on GPU memory.
	if renderer.fontTexture, err = renderer.sdlRenderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STATIC, int32(image.Width), int32(image.Height)); err != nil {
		panic(err)
	}

	pixels := unsafe.Slice((*byte)(image.Pixels), image.Width*image.Height*4)

	renderer.fontTexture.Update(nil, pixels, 4 * image.Width)
	renderer.fontTexture.SetBlendMode(sdl.BLENDMODE_BLEND)

	// Store our identifier
	io.Fonts().SetTextureID(imgui.TextureID(unsafe.Pointer(renderer.fontTexture)))
}

func (renderer *SDLRenderer) destroyDeviceObjects() {
	renderer.destroyFontsTexture()
}

func (renderer *SDLRenderer) createDeviceObjects() {
	renderer.createFontsTexture()
}

// PreRender causes the display buffer to be prepared for new output.
func (renderer *SDLRenderer) PreRender(clearColor [4]float32) {
	renderer.sdlRenderer.SetDrawColor(
		uint8(clearColor[0] * 255),
		uint8(clearColor[1] * 255),
		uint8(clearColor[2] * 255),
		uint8(clearColor[3] * 255),
	)
	renderer.sdlRenderer.Clear()
}

// Render draws the provided imgui draw data.
func (renderer *SDLRenderer) Render(displaySize [2]float32, framebufferSize [2]float32, drawData imgui.DrawData) {
	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := displaySize[0], displaySize[1]
	fbWidth, fbHeight := framebufferSize[0], framebufferSize[1]
	if (fbWidth <= 0) || (fbHeight <= 0) {
		return
	}
	//rsx, rsy := renderer.sdlRenderer.GetScale()
	render_scale := []float32{
		fbWidth / displayWidth,
		fbHeight / displayHeight,
	}

	// Backup state
	lastClipEnabled := renderer.sdlRenderer.IsClipEnabled()
	lastViewport := renderer.sdlRenderer.GetViewport()
	lastClipRect := renderer.sdlRenderer.GetClipRect()

	// Setup render state

	// Setup viewport
	clip_off := drawData.DisplayPos()
	clip_scale := imgui.Vec2{
		X: render_scale[0],
		Y: render_scale[1],
	}

	vtxSize, posVtx, uvVtx, colVtx := imgui.VertexBufferLayout()
	idxSize := imgui.IndexBufferLayout()

	// Draw
	for _, list := range drawData.CommandLists() {
		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		indexBuffer, _ := list.IndexBuffer()
		vertexBufferSize /= vtxSize

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				// Project scissor/clipping rectangles into framebuffer space
				clip_min := imgui.Vec2{
					X: (cmd.ClipRect().X - clip_off.X) * clip_scale.X,
					Y: (cmd.ClipRect().Y - clip_off.Y) * clip_scale.Y,
				}
				clip_max := imgui.Vec2{
					X: (cmd.ClipRect().Z - clip_off.X) * clip_scale.X,
					Y: (cmd.ClipRect().W - clip_off.Y) * clip_scale.Y,
				}

				if clip_min.X < 0 { clip_min.X = 0.0 }
				if clip_min.Y < 0 { clip_min.Y = 0.0 }
				if clip_max.X > fbWidth { clip_max.X = fbWidth }
				if clip_max.Y > fbHeight { clip_max.Y = fbHeight }
				if (clip_max.X <= clip_min.X) || (clip_max.Y <= clip_min.Y) {
					continue
				}

				r := sdl.Rect {
					X: int32(clip_min.X),
					Y: int32(clip_min.Y),
					W: int32(clip_max.X - clip_min.X),
					H: int32(clip_max.Y - clip_min.Y),
				}
				renderer.sdlRenderer.SetClipRect(&r)

				//nolint:unsafeptr
				texID := unsafe.Pointer(cmd.TextureID())
				tex := (*sdl.Texture)(texID)
				vtxOfs := unsafe.Add(vertexBuffer, (vtxSize * cmd.VertexOffset()))
				idxOfs := unsafe.Add(indexBuffer, (idxSize * cmd.IndexOffset()))
				xy := (*float32)(unsafe.Add(vtxOfs, posVtx))
				uv := (*float32)(unsafe.Add(vtxOfs, uvVtx))
				color := (*sdl.Color)(unsafe.Add(vtxOfs, colVtx))

				/*var indices interface{}
				switch idxSize {
				case 1:
					indices1 := unsafe.Slice((*byte)(idxOfs), cmd.ElementCount())
					indices = indices1
				case 2:
					indices2 := unsafe.Slice((*uint16)(idxOfs), cmd.ElementCount())
					indices = indices2
				case 4:
					indices4 := unsafe.Slice((*uint16)(idxOfs), cmd.ElementCount())
					indices = indices4
				default:
					panic(fmt.Errorf("unsupported idx size = %d",  idxSize))
				}
				*/

				_ = vertexBufferSize
				_ = tex
				_ = xy
				_ = uv
				_ = color

				num_vert := vertexBufferSize - cmd.VertexOffset()

				err := renderer.sdlRenderer.RenderGeometryRaw(tex,
					xy, vtxSize,
					color, vtxSize,
					uv, vtxSize,
					num_vert,
					idxOfs, cmd.ElementCount(), idxSize,
				)
				if err != nil {
					fmt.Println(err)
				}
			}
		}


	}

	// Restore modified State
	renderer.sdlRenderer.SetViewport(&lastViewport)
	if lastClipEnabled {
		renderer.sdlRenderer.SetClipRect(&lastClipRect)
	} else {
		renderer.sdlRenderer.SetClipRect(nil)
	}
}

func (renderer *SDLRenderer) PostRender() {
	renderer.sdlRenderer.Present()
}