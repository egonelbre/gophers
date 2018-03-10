package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

type Atlas struct {
	Frames []Frame
	Meta   Meta
}

type Frame struct {
	Frame struct {
		X, Y, W, H int
	}
}

type Meta struct {
	FrameTags []FrameTag
}

type FrameTag struct {
	Name string
	From int
}

var (
	folder      = flag.String("folder", "", "output folder")
	transparent = flag.Bool("transparent", true, "transparent background")
)

func handlePng(infile io.Reader, outfile io.Writer) error {

	return nil
}
func main() {
	flag.Parse()

	if flag.Arg(0) == "" || flag.Arg(1) == "" || flag.Arg(2) == "" {
		flag.Usage()
		os.Exit(1)
	}

	infile, err := os.Open(flag.Arg(0))
	check(err)
	defer infile.Close()

	atlasfile, err := os.Open(flag.Arg(1))
	check(err)
	defer atlasfile.Close()

	source, err := png.Decode(infile)
	check(err)

	var atlas Atlas
	check(json.NewDecoder(atlasfile).Decode(&atlas))

	for _, frametag := range atlas.Meta.FrameTags {
		frame := atlas.Frames[frametag.From].Frame

		target := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
		if !*transparent {
			draw.Draw(target, target.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
		}

		draw.Draw(target, target.Bounds(), source, image.Pt(frame.X, frame.Y), draw.Src)

		outname := filepath.Join(flag.Arg(2), "gopher-"+frametag.Name+".png")
		os.MkdirAll(filepath.Dir(outname), 0755)
		outfile, err := os.Create(outname)
		check(err)
		check(png.Encode(outfile, target))
		outfile.Close()
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %v\n", err)
		os.Exit(1)
	}
}
