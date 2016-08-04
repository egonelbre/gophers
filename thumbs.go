package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

var (
	colnum   = flag.Int("c", 8, "columns for images")
	cellsize = flag.Int("s", 128, "cell width/height")
	output   = flag.String("o", ".thumb", "output file")

	cell = 0
)

func openimg(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	m, _, err := image.Decode(file)
	return m, err
}

func savejpg(path string, m image.Image) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	path = path[:len(path)-len(filepath.Ext(path))] + ".jpg"
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, m, &jpeg.Options{Quality: 90})
}

func savepng(path string, m image.Image) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	path = path[:len(path)-len(filepath.Ext(path))] + ".png"
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, m)
}

func process(dir, path string) error {
	m, err := openimg(filepath.Join(dir, path))
	if err != nil {
		return err
	}

	size := m.Bounds().Size()
	if size.X > size.Y {
		size.X, size.Y = cell, size.Y*cell/size.X
	} else {
		size.X, size.Y = size.X*cell/size.Y, cell
	}

	dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), m, m.Bounds(), draw.Over, nil)

	log.Println(filepath.Join(*output, path))
	return savejpg(filepath.Join(*output, path), dst)
}

func main() {
	flag.Parse()
	cell = *cellsize

	dir := flag.Arg(0)
	if dir == "" {
		dir = "."
	}

	files := []string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && filepath.HasPrefix(path, ".") && path != "." && path != ".." {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".png" && filepath.Ext(path) != ".jpg" {
			return nil
		}
		if strings.Contains(path, ".sketch.") {
			return nil
		}

		files = append(files, filepath.Join(dir, path))
		return process(dir, path)
	})

	if err != nil {
		log.Println(err)
	}
}
