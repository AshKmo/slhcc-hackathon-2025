# AshKmodify
The best code editor I have in my GitHub account.

## Building
This program relies on SDL2 and SDL2\_ttf.

You can build a portable version by running `go build -o build/ -ldflags '-r ./'` then running `ldd build/ashkmodify` to list all the dependent libraries and then moving them into the build directory
