package builtin

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	xbmp "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"github.com/maccavelli/gobble-cli/tool"
)

// maxImageBase64 is 4.5 MB of base64, under the inline limit Pi uses
// (image-resize-core.ts). The comparison is strict, as that file's is.
const maxImageBase64 = int(4.5 * 1024 * 1024)

const (
	maxImageWidth  = 2000
	maxImageHeight = 2000
)

// ImageMIME is the image type sniff read uses (0036-MADR D8).
func ImageMIME(b []byte) string { return imageType(b) }

// readImage is a read of an image file. A model that does not take images
// gets the note. Otherwise the image is prepared (0036-MADR D8).
func readImage(name, mime string, src io.Reader, env tool.Env) (tool.Result, error) {
	if !env.Images {
		r := tool.TextResult(fmt.Sprintf("Read image file [%s]\n%s", mime, imageNotSent))
		r.Summary = summary(fmt.Sprintf("read %s (%s, not sent)", name, mime))
		return r, nil
	}
	data, err := io.ReadAll(io.LimitReader(src, maxImageBytes+1))
	if err != nil {
		return tool.Result{}, fmt.Errorf("read %s: %w", name, unwrapPath(err))
	}
	if len(data) > maxImageBytes {
		return tool.Result{}, fmt.Errorf("read %s: image is larger than %d MB", name, maxImageBytes>>20)
	}
	img, note, err := PrepareImage(data, mime, !env.NoAutoResize)
	if err != nil {
		return tool.Result{}, fmt.Errorf("read %s: %w", name, err)
	}
	shown := mime
	if img.MediaType != "" {
		shown = img.MediaType
	}
	text := "Read image file [" + shown + "]"
	if note != "" {
		text += "\n" + note
	}
	r := tool.TextResult(text)
	if len(img.Data) == 0 {
		r.Summary = summary(fmt.Sprintf("read %s (%s, not sent)", name, mime))
		return r, nil
	}
	r.Images = []tool.Image{img}
	r.Summary = summary(fmt.Sprintf("read %s (%s)", name, shown))
	return r, nil
}

// PrepareImage prepares data, of the sniffed or given MIME type, for a model
// that takes images. autoResize follows images.autoResize: off sends data
// unchanged when its base64 is under 4.5 MB, and the size note otherwise.
// On, EXIF orientation is applied and the result is the smaller of PNG and
// JPEG within 2000×2000 and 4.5 MB. A GIF sends its first frame. A decode
// failure is the error; a size failure is the note and an empty image
// (0036-MADR D8, Pi's image-resize-core.ts and image-process.ts).
func PrepareImage(data []byte, mime string, autoResize bool) (tool.Image, string, error) {
	mime = canonicalMIME(mime)
	if mime == "" {
		mime = imageType(data)
	}
	if mime == "" {
		return tool.Image{}, "", errors.New("unsupported image")
	}
	if !autoResize {
		if base64.StdEncoding.EncodedLen(len(data)) < maxImageBase64 {
			return tool.Image{MediaType: mime, Data: data}, "", nil
		}
		return tool.Image{}, tool.ImageTooLarge, nil
	}
	orient := exifOrientation(data)
	img, err := decodePrepared(data, mime)
	if err != nil {
		return tool.Image{}, "", err
	}
	img = applyOrientation(img, orient)
	ow, oh := img.Bounds().Dx(), img.Bounds().Dy()
	if passthrough(mime, orient, ow, oh, data) {
		return tool.Image{MediaType: mime, Data: data}, "", nil
	}
	tw, th := fit(ow, oh, maxImageWidth, maxImageHeight)
	out, outMIME, rw, rh, ok := shrink(img, tw, th)
	if !ok {
		return tool.Image{}, tool.ImageTooLarge, nil
	}
	return tool.Image{MediaType: outMIME, Data: out}, dimensionNote(ow, oh, rw, rh), nil
}

func passthrough(mime string, orient, w, h int, data []byte) bool {
	if orient != 1 || (mime != mimePNG && mime != mimeJPEG) {
		return false
	}
	return w <= maxImageWidth && h <= maxImageHeight && base64.StdEncoding.EncodedLen(len(data)) < maxImageBase64
}

func canonicalMIME(mime string) string {
	mime, _, _ = strings.Cut(mime, ";")
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime == "image/jpg" {
		return mimeJPEG
	}
	return mime
}

func decodePrepared(data []byte, mime string) (image.Image, error) {
	r := bytes.NewReader(data)
	var (
		img image.Image
		err error
	)
	switch mime {
	case mimeJPEG:
		img, err = jpeg.Decode(r)
	case mimePNG:
		img, err = png.Decode(r)
	case "image/gif":
		img, err = gif.Decode(r)
	case "image/bmp":
		img, err = xbmp.Decode(r)
	case "image/webp":
		img, err = webp.Decode(r)
	default:
		return nil, errors.New("unsupported image")
	}
	if err != nil {
		return nil, err
	}
	return img, nil
}

func dimensionNote(ow, oh, w, h int) string {
	scale := float64(ow) / float64(w)
	return fmt.Sprintf("[Image: original %dx%d, displayed at %dx%d. Multiply coordinates by %.2f to map to original image.]", ow, oh, w, h, scale)
}

func fit(w, h, maxW, maxH int) (int, int) {
	if w > maxW {
		h = int(math.Round(float64(h) * float64(maxW) / float64(w)))
		w = maxW
	}
	if h > maxH {
		w = int(math.Round(float64(w) * float64(maxH) / float64(h)))
		h = maxH
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func shrink(src image.Image, w, h int) (data []byte, mime string, rw, rh int, ok bool) {
	for {
		data, mime, ok = encodeFit(src, w, h)
		if ok {
			return data, mime, w, h, true
		}
		if w == 1 && h == 1 {
			return nil, "", 0, 0, false
		}
		nw, nh := shrinkDim(w), shrinkDim(h)
		if nw == w && nh == h {
			return nil, "", 0, 0, false
		}
		w, h = nw, nh
	}
}

func shrinkDim(n int) int {
	if n == 1 {
		return 1
	}
	n = int(math.Floor(float64(n) * 0.75))
	if n < 1 {
		return 1
	}
	return n
}

func encodeFit(src image.Image, w, h int) ([]byte, string, bool) {
	scaled := src
	if src.Bounds().Dx() != w || src.Bounds().Dy() != h || src.Bounds().Min != (image.Point{}) {
		scaled = scaleTo(src, w, h)
	}
	var best []byte
	bestMIME := ""
	consider := func(data []byte, mime string) {
		if base64.StdEncoding.EncodedLen(len(data)) >= maxImageBase64 {
			return
		}
		if best == nil || len(data) < len(best) {
			best, bestMIME = data, mime
		}
	}
	if p, err := encodePNG(scaled); err == nil {
		consider(p, mimePNG)
	}
	for _, q := range []int{80, 85, 70, 55, 40} {
		if j, err := encodeJPEG(scaled, q); err == nil {
			consider(j, mimeJPEG)
		}
	}
	return best, bestMIME, best != nil
}

func scaleTo(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func applyOrientation(src image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flip(src, true, false)
	case 3:
		return flip(src, true, true)
	case 4:
		return flip(src, false, true)
	case 5:
		return flip(rotate90(src, true), true, false)
	case 6:
		return rotate90(src, true)
	case 7:
		return flip(rotate90(src, false), true, false)
	case 8:
		return rotate90(src, false)
	default:
		return src
	}
}

// rotate90 turns src. clockwise uses Pi's orientation-6 map; otherwise the
// orientation-8 map.
func rotate90(src image.Image, clockwise bool) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := range h {
		for x := range w {
			nx, ny := y, w-1-x
			if clockwise {
				nx, ny = h-1-y, x
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flip(src image.Image, horizontal, vertical bool) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			nx, ny := x, y
			if horizontal {
				nx = w - 1 - x
			}
			if vertical {
				ny = h - 1 - y
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
