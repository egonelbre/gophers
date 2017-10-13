package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	"os"

	"golang.org/x/image/draw"
)

var (
	width       = flag.Int("width", 506, "target min width")
	height      = flag.Int("height", 0, "target min height")
	repeat      = flag.Int("repeat", 3, "repeat count")
	transparent = flag.Bool("transparent", false, "transparent background")
	duplicate   = flag.Bool("duplicate", false, "use duplication instead of repeating animation")
)

func main() {
	flag.Parse()

	if flag.Arg(0) == "" || flag.Arg(1) == "" {
		flag.Usage()
		os.Exit(1)
	}

	srcfile, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer srcfile.Close()

	targetfile, err := os.Create(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer targetfile.Close()

	source, err := gif.DecodeAll(srcfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
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
		draw.Draw(d, m.Bounds().Add(offset), m, image.ZP, draw.Src)

		if *duplicate {
			target.Image = append(target.Image, d)
			target.Delay = append(target.Delay, source.Delay[i]/2)
			if len(source.Disposal) > 0 {
				target.Disposal = append(target.Disposal, source.Disposal[i])
			}
			if i != len(source.Image)-1 {
				target.Image = append(target.Image, d)
				target.Delay = append(target.Delay, source.Delay[i]/2)
				if len(source.Disposal) > 0 {
					target.Disposal = append(target.Disposal, source.Disposal[i])
				}
			}
		} else {
			target.Image = append(target.Image, d)
			target.Delay = append(target.Delay, source.Delay[i])
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

	err = gif.EncodeAll(targetfile, &target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode: %v\n", err)
		os.Exit(1)
	}
}
