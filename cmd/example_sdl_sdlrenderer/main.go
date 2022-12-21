//go:build sdl
// +build sdl

package main

import (
	"fmt"
	"os"

	"github.com/inkyblackness/imgui-go/v4"

	"github.com/inkyblackness/imgui-go-examples/internal/example"
	"github.com/inkyblackness/imgui-go-examples/internal/platforms"
	"github.com/inkyblackness/imgui-go-examples/internal/renderers"
)

func main() {
	context := imgui.CreateContext(nil)
	defer context.Destroy()
	io := imgui.CurrentIO()

	platform, err := platforms.NewSDL(io, platforms.SDLClientAPISDLRenderer)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
	defer platform.Dispose()

	rend, err := platform.CreateRenderer()
	if err != nil {
					panic(err)
	}
	sdlRenderer, err := renderers.NewSDLRenderer(rend)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
	defer sdlRenderer.Dispose()

	example.Run(platform, sdlRenderer)
}
