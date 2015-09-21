package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"image"
	"image/color"

	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

var (
	colnum   = flag.Int("c", 8, "columns for images")
	cellsize = flag.Int("s", 128, "cell width/height")
	output   = flag.String("o", "collage.jpg", "output file")
)

func main() {
	flag.Parse()

	dir := flag.Arg(0)
	if dir == "" {
		dir = "."
	}

	files := []string{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if (filepath.Ext(path) != ".png" && filepath.Ext(path) != ".jpg") || path == *output {
			return nil
		}

		files = append(files, filepath.Join(dir, path))
		return nil
	})

	ordered := []image.Image{}
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			log.Println(path, err)
			continue
		}
		m, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			continue
		}
		sz := m.Bounds().Size()
		if sz.X < 64 || sz.Y < 64 {
			continue
		}
		ordered = append(ordered, m)
	}

	images := make([]image.Image, len(ordered))
	for src, dst := range rand.Perm(len(ordered)) {
		images[dst] = ordered[src]
	}

	cols := *colnum
	cell := *cellsize
	rows := (len(images) + cols - 1) / cols
	dst := image.NewRGBA(image.Rect(0, 0, cell*cols, cell*rows))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}), image.Point{}, draw.Src)

	for i, m := range images {
		col := i % cols
		row := i / cols

		sz := m.Bounds().Size()
		dz := sz
		if sz.X > sz.Y {
			dz.X = cell
			dz.Y = cell * sz.Y / sz.X
		} else {
			dz.Y = cell
			dz.X = cell * sz.X / sz.Y
		}

		z := image.Point{cell * col, cell * row}
		r := image.Rectangle{
			Min: z,
			Max: z.Add(dz),
		}
		r = r.Add(image.Point{cell / 2, cell / 2}).
			Sub(image.Point{dz.X / 2, dz.Y / 2})

		draw.CatmullRom.Scale(dst, r, m, m.Bounds(), draw.Over, nil)
	}

	result, err := os.Create(*output)
	if err != nil {
		log.Println(err)
		return
	}

	switch filepath.Ext(*output) {
	case ".png":
		if err := png.Encode(result, dst); err != nil {
			log.Println(err)
			return
		}
	case ".jpg":
		if err := jpeg.Encode(result, dst, &jpeg.Options{Quality: 90}); err != nil {
			log.Println(err)
			return
		}
	}
}
