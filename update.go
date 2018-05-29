package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

const (
	ThumbnailSize = 128
	MaxColumns    = 6

	InkscapePath = `c:\Program Files\Inkscape\inkscape.exe`
)

const README_HEADER = `
# Gophers....

The Go gopher was designed by the awesome [Renee French](http://reneefrench.blogspot.com/). Read http://blog.golang.org/gopher for more details.

The images and art-work in this repository are under [CC0 license](https://creativecommons.org/publicdomain/zero/1.0/).

However, if you do use something, you are encouraged to:

* tweet about the used, remixed or printed result @egonelbre
* submit new ideas via twitter @egonelbre
* request some sketch to be vectorized

Or if you like to directly support me:

<a target="_blank" href="https://www.buymeacoffee.com/egon"><img alt="Buy me a Coffee" src=".thumb/animation/buy-morning-coffee-3x.gif"></a>

<img src=".thumb/icon/emoji-3x.png ">

<img src=".thumb/animation/gopher-dance-long-3x.gif "> <img src=".thumb/icon/gotham-3x.png ">

<img src=".thumb/animation/2bit-sprite/demo.gif ">

`

const VECTOR_HEADER = `
# Vector

Here are svg images that can be modified for your own needs.

`

const SKETCHES_HEADER = `
# Sketches

Here are several hand-drawn images. Let me know if you would like to
see a particular one be vectorized.

`

type ImageLink struct {
	Thumb  string
	Actual string
	Bounds image.Rectangle
}

type Collage struct {
	Image *image.RGBA
	X, Y  int

	ColumnsPerRow int
	CellSize      int

	Name   string
	Output string
	Folder string
	Links  []ImageLink
}

func NewCollage(count, columnsPerRow, cellSize int) *Collage {
	if count < columnsPerRow {
		columnsPerRow = count
	}

	rowCount := (count + columnsPerRow - 1) / columnsPerRow
	bounds := image.Rect(0, 0, columnsPerRow*cellSize, rowCount*cellSize)
	collage := &Collage{
		Image: image.NewRGBA(bounds),
		X:     0, Y: 0,
		ColumnsPerRow: columnsPerRow,
		CellSize:      cellSize,
	}

	draw.Draw(collage.Image, collage.Image.Bounds(), image.White, image.ZP, draw.Src)

	return collage
}

func (collage *Collage) Bounds(x, y int) image.Rectangle {
	x0 := x * collage.CellSize
	y0 := y * collage.CellSize
	return image.Rect(x0, y0, x0+collage.CellSize, y0+collage.CellSize)
}

func (collage *Collage) Draw(path string, m image.Image) {
	frame := collage.Bounds(collage.X, collage.Y)
	collage.Links = append(collage.Links, ImageLink{
		Actual: path,
		Bounds: frame,
	})

	inner := FitBoundsIntoFrame(m.Bounds(), frame)
	draw.CatmullRom.Scale(collage.Image, inner, m, m.Bounds(), draw.Over, nil)

	collage.X++
	if collage.X >= collage.ColumnsPerRow {
		collage.X = 0
		collage.Y++
	}
}

func MakeCollage(name, folder, output string) *Collage {
	log.Printf("Creating collage\n")
	log.Printf("> name  : %v\n", name)
	log.Printf("> folder: %v\n", folder)
	log.Printf("> save  : %v\n", output)

	files, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Printf("> ERROR: %v\n", err)
		return nil
	}

	if len(files) == 0 {
		log.Printf("> error: no files\n")
		return nil
	}

	sort.Sort(FileInfos(files))

	collage := NewCollage(len(files), MaxColumns, ThumbnailSize)
	collage.Name = name
	collage.Output = output
	collage.Folder = folder
	for _, file := range files {
		path := filepath.Join(folder, file.Name())
		log.Printf("> add: %v\n", path)
		m, err := LoadImage(path)
		if err != nil {
			log.Printf("> error: %v\n", err)
			continue
		}

		collage.Draw(path, m)
	}

	if err := SaveImage(collage.Image, output); err != nil {
		log.Printf("> ERROR: %v\n", err)
	}

	return collage
}

type Thumbs struct {
	Size   int
	Name   string
	Output string
	Folder string
	Links  []ImageLink
}

func (thumbs *Thumbs) ExportSVG(actual, out string) {
	os.MkdirAll(filepath.Dir(out), 0755)
	os.Remove(out)

	// inkscape -h 128 -e hiking.png hiking.svg
	cmd := exec.Command(InkscapePath,
		"-h", strconv.Itoa(thumbs.Size),
		"-e", out,
		actual)
	cmd.Run()

	thumbs.Links = append(thumbs.Links, ImageLink{
		Actual: actual,
		Thumb:  out,
	})
}

func (thumbs *Thumbs) Downscale(actual, out string, m image.Image) image.Image {
	targetSize := image.Point{0, thumbs.Size}
	targetSize.X = m.Bounds().Dx() * thumbs.Size / m.Bounds().Dy()
	inner := image.Rectangle{image.ZP, targetSize}

	thumbs.Links = append(thumbs.Links, ImageLink{
		Actual: actual,
		Thumb:  out,
		Bounds: inner,
	})

	rgba := image.NewRGBA(inner)
	draw.CatmullRom.Scale(rgba, rgba.Bounds(), m, m.Bounds(), draw.Over, nil)

	return rgba
}

func MakeThumbs(name, folder, output string) *Thumbs {
	log.Printf("Creating thumbs\n")
	log.Printf("> name  : %v\n", name)
	log.Printf("> folder: %v\n", folder)
	log.Printf("> save  : %v\n", output)

	files, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Printf("> ERROR: %v\n", err)
		return nil
	}

	if len(files) == 0 {
		log.Printf("> error: no files\n")
		return nil
	}

	sort.Sort(FileInfos(files))

	thumbs := &Thumbs{}
	thumbs.Size = ThumbnailSize
	thumbs.Name = name
	thumbs.Output = output
	thumbs.Folder = folder

	for _, file := range files {
		if strings.Contains(file.Name(), ".sheet.") {
			continue
		}

		path := filepath.Join(folder, file.Name())
		log.Printf("> add: %v\n", path)

		outpath := filepath.Join(output, file.Name())
		outpath = ReplaceExt(outpath, ".png")

		if filepath.Ext(path) == ".svg" {
			thumbs.ExportSVG(path, outpath)
			continue
		}

		m, err := LoadImage(path)
		if err != nil {
			log.Printf("> error: %v\n", err)
			continue
		}

		out := thumbs.Downscale(path, outpath, m)
		if err := SavePNG(out, outpath); err != nil {
			log.Printf("> error: %v\n", err)
			continue
		}
	}

	return thumbs
}

func main() {
	dirs, _ := ioutil.ReadDir("sketch")
	sort.Sort(FileInfos(dirs))

	sketches := []*Thumbs{}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		thumbs := MakeThumbs(
			strings.Title(dir.Name()),
			filepath.Join("sketch", dir.Name()),
			filepath.Join(".thumb", "sketch", dir.Name()))

		if thumbs != nil {
			sketches = append(sketches, thumbs)
		}
	}

	dirs, _ = ioutil.ReadDir("vector")
	sort.Sort(FileInfos(dirs))

	vectors := []*Thumbs{}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		thumbs := MakeThumbs(
			strings.Title(dir.Name()),
			filepath.Join("vector", dir.Name()),
			filepath.Join(".thumb", "vector", dir.Name()))

		if thumbs != nil {
			vectors = append(vectors, thumbs)
		}
	}

	file, err := os.Create("README.md")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fmt.Fprintf(file, "%v\n", README_HEADER)

	fmt.Fprintf(file, "%v\n", VECTOR_HEADER)
	file.Write(CreateThumbsIndex(false, vectors))

	fmt.Fprintf(file, "%v\n", SKETCHES_HEADER)
	file.Write(CreateThumbsIndex(false, sketches))
}

func CreateThumbsIndex(withtitle bool, thumbsets []*Thumbs) []byte {
	var buf bytes.Buffer

	for _, thumbs := range thumbsets {
		if withtitle {
			fmt.Fprintf(&buf, "\n### [%v](%v)\n\n",
				thumbs.Name,
				filepath.ToSlash(thumbs.Folder))
		}

		for _, thumb := range thumbs.Links {
			fmt.Fprintf(&buf, "[<img src=\"%v\">](%v)\n",
				filepath.ToSlash(thumb.Thumb),
				filepath.ToSlash(thumb.Actual))
		}
	}
	fmt.Fprintf(&buf, "\n\n")

	return buf.Bytes()
}

func CreateCollageIndex(withtitle bool, collages []*Collage) []byte {
	var buf bytes.Buffer
	for _, collage := range collages {
		if withtitle {
			fmt.Fprintf(&buf, "\n### [%v](%v)\n\n",
				collage.Name,
				filepath.ToSlash(collage.Folder))
		}

		fmt.Fprintf(&buf, "[<img src=\"%v\">](%v)\n",
			filepath.ToSlash(collage.Output),
			filepath.ToSlash(collage.Folder))

		/*
			fmt.Fprintf(&buf, "<div>\n")
			fmt.Fprintf(&buf, "  <img src=\"%v\" usemap=\"#%v\" />\n", , collage.Name)
			fmt.Fprintf(&buf, "  <map name=\"%v\">\n", collage.Name)
			for _, link := range collage.Links {
				r := link.Bounds
				fmt.Fprintf(&buf, "    <area shape=\"rect\" ")
				fmt.Fprintf(&buf, "coords=\"%v,%v,%v,%v\" ", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
				fmt.Fprintf(&buf, "href=\"%v\" ", filepath.ToSlash(link.Actual))
				fmt.Fprintf(&buf, ">\n")
			}
			fmt.Fprintf(&buf, "  </map>\n")
			fmt.Fprintf(&buf, "</div>\n\n")
		*/
	}

	return buf.Bytes()
}

/* geometry */

func FitBoundsIntoFrame(bounds, frame image.Rectangle) image.Rectangle {
	size := bounds.Size()
	frameSize := frame.Size()
	targetSize := image.Point{}

	aspect := float64(size.X) / float64(size.Y)

	if aspect < 1.0 { // x is smaller
		targetSize.X = frameSize.Y * size.X / size.Y
		targetSize.Y = frameSize.Y
	} else { // y is smaller
		targetSize.X = frameSize.X
		targetSize.Y = frameSize.X * size.Y / size.X
	}

	frameCenter := frame.Min.Add(frameSize.Div(2))
	x0 := frameCenter.X - targetSize.X/2
	x1 := frameCenter.X + targetSize.X/2

	y0 := frame.Max.Y - targetSize.Y
	y1 := frame.Max.Y

	return image.Rectangle{
		Min: image.Point{x0, y0},
		Max: image.Point{x1, y1},
	}
}

/* basic file utilties */

type FileInfos []os.FileInfo

func (xs FileInfos) Len() int      { return len(xs) }
func (xs FileInfos) Swap(i, k int) { xs[i], xs[k] = xs[k], xs[i] }
func (xs FileInfos) Less(i, k int) bool {
	return xs[i].Name() < xs[k].Name()
}

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	m, _, err := image.Decode(file)
	return m, err
}

func ReplaceExt(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

func SaveImage(m image.Image, path string) error {
	switch filepath.Ext(path) {
	case ".jpg":
		return SaveJPG(m, path)
	case ".png":
		return SavePNG(m, path)
	}
	return errors.New("unknown output format")
}

func SaveJPG(m image.Image, path string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	path = ReplaceExt(path, ".jpg")

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return jpeg.Encode(file, m, &jpeg.Options{Quality: 90})
}

func SavePNG(m image.Image, path string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	path = ReplaceExt(path, ".png")

	path = path[:len(path)-len(filepath.Ext(path))] + ".png"
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, m)
}
