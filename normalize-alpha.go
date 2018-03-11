package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func handleFile(name string) error {
	var source *image.NRGBA
	{
		file, err := os.Open(name)
		if err != nil {
			return err
		}

		m, err := png.Decode(file)
		if err != nil {
			file.Close()
			return err
		}
		file.Close()

		if rgba, ok := m.(*image.NRGBA); !ok {
			return errors.New("not RGBA")
		} else {
			source = rgba
		}
	}

	background := [4]uint8{0, 0, 0, 0}
	if source.Pix[3] == 0 {
		copy(background[:], source.Pix[:4])
	}

	for i := 0; i < len(source.Pix); i += 4 {
		if source.Pix[i+3] == 0 {
			copy(source.Pix[i:i+4], background[:])
		}
	}

	{
		file, err := os.Create(name)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := png.Encode(file, source); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(1)
	}

	matches, err := filepath.Glob(flag.Arg(0))
	check(err)
	for _, match := range matches {
		err := handleFile(match)
		if err != nil {
			fmt.Printf("%v: %v\n", match, err)
		}
	}
	/*
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
	*/
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %v\n", err)
		os.Exit(1)
	}
}
