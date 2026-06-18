package card

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/agenticraptor/readme-radar/internal/report"
)

// PNG renders the verdict card to w as a PNG, at the given integer scale
// (1 = 860×470). A scale of 2 produces a crisp, social-ready image.
func PNG(rep report.Report, w io.Writer, scale int) error {
	if scale < 1 {
		scale = 1
	}
	m := build(rep)
	r := &raster{scale: scale, img: image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))}
	if err := r.loadFonts(); err != nil {
		return err
	}
	r.draw(m)
	return png.Encode(w, r.img)
}

type raster struct {
	scale int
	img   *image.RGBA
	reg   *opentype.Font
	bold  *opentype.Font
}

func (r *raster) loadFonts() error {
	var err error
	if r.reg, err = opentype.Parse(goregular.TTF); err != nil {
		return fmt.Errorf("load regular font: %w", err)
	}
	if r.bold, err = opentype.Parse(gobold.TTF); err != nil {
		return fmt.Errorf("load bold font: %w", err)
	}
	return nil
}

func (r *raster) draw(m model) {
	accent := hexRGBA(m.accent)

	// Background + card + accent rail.
	r.fillRound(0, 0, width, height, 20, hexRGBA(bgColor))
	r.fillRound(14, 14, width-28, height-28, 16, hexRGBA(cardColor))
	r.fillRound(14, 14, 8, height-28, 4, accent)

	// Header.
	r.radar(50, 54, accent)
	r.text(78, 62, 28, true, hexRGBA(textColor), "readme-radar")
	r.textRight(width-40, 62, 18, false, hexRGBA(dimColor), trunc(m.target, 52))
	r.hline(40, width-40, 86, hexRGBA(strokeColor))

	// Grade badge.
	r.fillRound(44, 112, 100, 100, 18, accent)
	r.textCenter(94, 182, 68, true, hexRGBA(bgColor), m.grade)

	// Verdict + score + facts.
	r.text(166, 146, 30, true, accent, m.verdict)
	r.text(166, 178, 20, false, hexRGBA(dimColor), fmt.Sprintf("%d / 100", m.score))
	r.text(166, 206, 16, false, hexRGBA(dimColor), m.facts)

	// Axis bars.
	y := 256
	for _, a := range m.axes {
		r.text(44, y+4, 16, false, hexRGBA(textColor), a.name)
		r.fillRound(180, y-6, 500, 12, 6, hexRGBA(trackColor))
		fw := 500 * clamp(a.score) / 100
		if fw > 0 {
			r.fillRound(180, y-6, fw, 12, 6, hexRGBA(a.color))
		}
		r.text(700, y+4, 15, true, hexRGBA(a.color), a.grade)
		r.text(724, y+4, 15, false, hexRGBA(dimColor), fmt.Sprintf("%d", a.score))
		y += 34
	}

	// Narrative.
	ny := y + 14
	for _, line := range m.summary {
		r.text(44, ny, 15, false, hexRGBA(textColor), line)
		ny += 22
	}

	// Footer.
	r.text(44, height-26, 13, false, hexRGBA(dimColor), m.footer)
	r.textRight(width-40, height-26, 13, false, hexRGBA(dimColor),
		fmt.Sprintf("grade %s · %s", m.grade, lower(m.verdict)))
}

// --- primitives (logical coords; scaled internally) ---

func (r *raster) face(size float64, bold bool) font.Face {
	f := r.reg
	if bold {
		f = r.bold
	}
	face, _ := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size * float64(r.scale),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face
}

func (r *raster) text(x, y int, size float64, bold bool, c color.Color, s string) {
	face := r.face(size, bold)
	defer face.Close()
	d := &font.Drawer{Dst: r.img, Src: image.NewUniform(c), Face: face,
		Dot: fixed.P(x*r.scale, y*r.scale)}
	d.DrawString(s)
}

func (r *raster) textRight(x, y int, size float64, bold bool, c color.Color, s string) {
	face := r.face(size, bold)
	defer face.Close()
	w := font.MeasureString(face, s).Round()
	d := &font.Drawer{Dst: r.img, Src: image.NewUniform(c), Face: face,
		Dot: fixed.Point26_6{X: fixed.I(x*r.scale) - fixed.I(w), Y: fixed.I(y * r.scale)}}
	d.DrawString(s)
}

func (r *raster) textCenter(x, y int, size float64, bold bool, c color.Color, s string) {
	face := r.face(size, bold)
	defer face.Close()
	w := font.MeasureString(face, s).Round()
	d := &font.Drawer{Dst: r.img, Src: image.NewUniform(c), Face: face,
		Dot: fixed.Point26_6{X: fixed.I(x*r.scale) - fixed.I(w)/2, Y: fixed.I(y * r.scale)}}
	d.DrawString(s)
}

func (r *raster) fillRound(x, y, w, h, rad int, c color.Color) {
	s := r.scale
	x, y, w, h, rad = x*s, y*s, w*s, h*s, rad*s
	if rad*2 > h {
		rad = h / 2
	}
	if rad*2 > w {
		rad = w / 2
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if r.inRound(i, j, w, h, rad) {
				r.img.Set(x+i, y+j, c)
			}
		}
	}
}

func (r *raster) inRound(i, j, w, h, rad int) bool {
	if rad <= 0 {
		return true
	}
	// corner centers
	type pt struct{ cx, cy int }
	corners := []pt{{rad, rad}, {w - rad, rad}, {rad, h - rad}, {w - rad, h - rad}}
	left, right, top, bottom := i < rad, i >= w-rad, j < rad, j >= h-rad
	for _, cn := range corners {
		inX := (cn.cx == rad && left) || (cn.cx == w-rad && right)
		inY := (cn.cy == rad && top) || (cn.cy == h-rad && bottom)
		if inX && inY {
			dx := float64(i - cn.cx + boolI(i >= cn.cx))
			dy := float64(j - cn.cy + boolI(j >= cn.cy))
			return dx*dx+dy*dy <= float64(rad*rad)
		}
	}
	return true
}

func (r *raster) hline(x1, x2, y int, c color.Color) {
	s := r.scale
	for x := x1 * s; x < x2*s; x++ {
		r.img.Set(x, y*s, c)
	}
}

func (r *raster) radar(cx, cy int, c color.RGBA) {
	for i, rad := range []int{16, 11, 6} {
		a := uint8(110 + 50*i)
		ring := color.RGBA{c.R, c.G, c.B, a}
		r.strokeCircle(cx, cy, rad, ring)
	}
	r.fillDisc(cx, cy, 3, c)
	// sweep line
	s := r.scale
	for t := 0; t < 14*s; t++ {
		r.img.Set(cx*s+t, cy*s-t, c)
		r.img.Set(cx*s+t+1, cy*s-t, c)
	}
}

func (r *raster) strokeCircle(cx, cy, rad int, c color.Color) {
	s := r.scale
	cxs, cys, rs := cx*s, cy*s, float64(rad*s)
	for deg := 0; deg < 360*2; deg++ {
		a := float64(deg) / 2 * math.Pi / 180
		for w := -s; w <= s; w++ {
			rr := rs + float64(w)*0.6
			x := cxs + int(rr*math.Cos(a))
			yy := cys + int(rr*math.Sin(a))
			r.img.Set(x, yy, c)
		}
	}
}

func (r *raster) fillDisc(cx, cy, rad int, c color.Color) {
	s := r.scale
	cxs, cys, rs := cx*s, cy*s, rad*s
	for j := -rs; j <= rs; j++ {
		for i := -rs; i <= rs; i++ {
			if i*i+j*j <= rs*rs {
				r.img.Set(cxs+i, cys+j, c)
			}
		}
	}
}

func boolI(b bool) int {
	if b {
		return 0
	}
	return 1
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
