# AshKmodify
The best code editor I have in my GitHub account.

## Building
`go build`

## Running
This program relies on SDL2, SDL2\_ttf and SDL2\_img.

## A confession
Unfortunately I encountered a memory leak that occurred as the main window was resized, and, having no idea how to fix it, resorted to ChatGPT with the prompt "spot the memory leak" and the file `main.go`. The issue turned out to be that I had not properly created a renderer for the window but had somehow still managed to render things to it without any errors. All that was needed was to replace `window.GetRenderer()` with `sdl.CreateRenderer()`. Bother.
