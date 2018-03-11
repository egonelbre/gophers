package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

var (
	width       = flag.Int("width", 506, "target min width")
	height      = flag.Int("height", 128, "target min height")
	repeat      = flag.Int("repeat", 3, "repeat count")
	transparent = flag.Bool("transparent", false, "transparent background")
	duplicate   = flag.Bool("duplicate", false, "use duplication instead of repeating animation")
	duration    = flag.Int("duration", 0, "override frame duration")
)

func handleGif(infile io.Reader, outfile io.Writer) error {
	// TODO: fix handling of different disposals

	source, err := gif.DecodeAll(infile)
	if err != nil {
		return fmt.Errorf("failed to decode gif: %v", err)
	}

	size := image.Pt(source.Config.Width, source.Config.Height)
	if size.X < *width {
		size.X = *width
	}
	if size.Y < *height {
		size.Y = *height
	}

	var target gif.GIF
	target.Config = source.Config
	target.Config.Width = size.X
	target.Config.Height = size.Y

	target.LoopCount = source.LoopCount
	target.BackgroundIndex = source.BackgroundIndex

	offset := image.Pt(
		size.X/2-source.Config.Width/2,
		size.Y/2-source.Config.Height/2,
	)

	for i, m := range source.Image {
		d := image.NewPaletted(image.Rectangle{image.ZP, size}, m.Palette)
		if !*transparent {
			for k := range d.Pix {
				d.Pix[k] = m.Pix[0]
			}
		}
		draw.Draw(d, m.Bounds().Add(offset), m, image.ZP, draw.Over)

		delay := source.Delay[i]
		if *duration > 0 {
			delay = *duration
		}

		if *duplicate {
			target.Image = append(target.Image, d)
			target.Delay = append(target.Delay, delay/2)
			if len(source.Disposal) > 0 {
				target.Disposal = append(target.Disposal, source.Disposal[i])
			}
			if i != len(source.Image)-1 {
				target.Image = append(target.Image, d)
				target.Delay = append(target.Delay, delay/2)
				if len(source.Disposal) > 0 {
					target.Disposal = append(target.Disposal, source.Disposal[i])
				}
			}
		} else {
			target.Image = append(target.Image, d)
			target.Delay = append(target.Delay, delay)
			if len(source.Disposal) > 0 {
				target.Disposal = append(target.Disposal, source.Disposal[i])
			}
		}
	}

	if !*duplicate {
		n := len(source.Image)
		for k := 1; k < *repeat; k++ {
			for i := 0; i < n; i++ {
				target.Image = append(target.Image, target.Image[i])
				target.Delay = append(target.Delay, target.Delay[i])
				if len(target.Disposal) > 0 {
					target.Disposal = append(target.Disposal, target.Disposal[i])
				}
			}
		}
	}

	err = gif.EncodeAll(outfile, &target)
	if err != nil {
		return fmt.Errorf("failed to encode to gif: %v", err)
	}

	return nil
}

func handlePng(infile io.Reader, outfile io.Writer) error {
	source, err := png.Decode(infile)
	if err != nil {
		return fmt.Errorf("failed to decode png: %v", err)
	}

	size := source.Bounds().Size()
	if size.X < *width {
		size.X = *width
	}
	if size.Y < *height {
		size.Y = *height
	}

	offset := image.Pt(
		size.X/2-source.Bounds().Dx()/2,
		size.Y/2-source.Bounds().Dy()/2,
	)

	target := image.NewRGBA(image.Rectangle{image.ZP, size})
	if !*transparent {
		draw.Draw(target, target.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Over)
	}
	draw.Draw(target, source.Bounds().Add(offset), source, image.ZP, draw.Over)

	err = png.Encode(outfile, target)
	if err != nil {
		return fmt.Errorf("failed to encode to png: %v", err)
	}

	return nil
}
func main() {
	flag.Parse()

	if flag.Arg(0) == "" || flag.Arg(1) == "" {
		flag.Usage()
		os.Exit(1)
	}

	infile, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer infile.Close()

	outfile, err := os.Create(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer outfile.Close()

	switch filepath.Ext(flag.Arg(0)) {
	case ".gif":
		err := handleGif(infile, outfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed twitterifying gif: %v\n", err)
			os.Exit(1)
		}
	case ".png":
		err := handlePng(infile, outfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed twitterifying png: %v\n", err)
			os.Exit(1)
		}
	}
}
