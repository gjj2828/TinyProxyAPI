// tools/genicon 生成托盘图标 icon.ico（纯标准库，一次性/可重复生成器）。
//
// 用法：go run ./tools/genicon
//
// 产出（写在仓库根目录，即运行工作目录的父目录）：
//   - icon.ico                ICO 容器，含 16 / 32 / 48 / 256 四个尺寸
//   - .superpowers/sdd/icon-preview.png  256px 预览，供人眼检查（已 gitignore）
//
// 图形：一进三出的分叉路由。底部一根入线向上接到分叉点，分叉成三根出线
// 向右延伸，四个端点各有一个圆点。笔画色 #10b981，背景全透明。
// 绘制在 4 倍分辨率的缓冲上，再按 2x2 平均降采样得到抗锯齿。
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	strokeColor = "#10b981"
	supersample = 4
)

// sizes 是要产出的四个尺寸，按升序排列。
var sizes = []int{16, 32, 48, 256}

// strokeRGB 是笔画颜色的三通道分量。#10b981 = (16, 185, 129)
const (
	strokeR = 0x10
	strokeG = 0xB9
	strokeB = 0x81
)

func main() {
	r, g, b := strokeR, strokeG, strokeB
	fmt.Printf("genicon: stroke %s = rgb(%d,%d,%d)\n", strokeColor, r, g, b)

	dirs, blobs := buildICO()

	out := filepath.Join(repoRoot(), "icon.ico")
	if err := writeICO(out, dirs, blobs); err != nil {
		fmt.Fprintf(os.Stderr, "genicon: write %s: %v\n", out, err)
		os.Exit(1)
	}
	fmt.Printf("genicon: wrote %s (%d bytes)\n", out, len(icoBytes(dirs, blobs)))

	// 预览图：用 256 尺寸的 RGBA 原图另存一份，便于肉眼检查
	if err := writePreview("icon-preview.png", renderGlyph(256, r, g, b)); err != nil {
		fmt.Fprintf(os.Stderr, "genicon: write preview: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("genicon: wrote .superpowers/sdd/icon-preview.png")

	verifyICO(dirs, blobs)
}

// iconDir 是 ICO 目录中的一条条目（64 位平台不要求对齐）。
type iconDir struct {
	wWidth, hHeight byte
	colorCount      byte
	reserved        byte
	planes          uint16
	bitCount        uint16
	bytesInRes      uint32
	imageOffset     uint32
}

// buildICO 渲染四个尺寸并组装目录条目与数据块。
func buildICO() ([]iconDir, [][]byte) {
	r, g, b := strokeR, strokeG, strokeB
	dirs := make([]iconDir, 0, len(sizes))
	blobs := make([][]byte, 0, len(sizes))

	// 数据块依次追加；偏移从 ICO 头(6) + 目录(16*N) 之后开始计算
	const headerSize = 6
	offset := uint32(headerSize + 16*len(sizes))
	for _, s := range sizes {
		img := renderGlyph(s, r, g, b)
		var blob []byte
		var bitCount uint16
		if s >= 256 {
			// 256 用 PNG 内嵌。ICO 目录里 0 表示 256。
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				panic(err)
			}
			blob = buf.Bytes()
			bitCount = 0 // 未使用；PNG 自带色彩深度
		} else {
			blob = encodeBMP(img, s)
			bitCount = 32
		}

		dirs = append(dirs, iconDir{
			wWidth:      widthByte(s),
			hHeight:     widthByte(s),
			colorCount:  0,
			reserved:    0,
			planes:      1,
			bitCount:    bitCount,
			bytesInRes:  uint32(len(blob)),
			imageOffset: offset,
		})
		blobs = append(blobs, blob)
		offset += uint32(len(blob))
	}
	return dirs, blobs
}

// widthByte 把尺寸映射到 ICO 目录字段：256 填 0。
func widthByte(s int) byte {
	if s >= 256 {
		return 0
	}
	return byte(s)
}

// encodeBMP 把 RGBA 图像编码为 ICO 使用的 32bpp 未压缩 DIB：
// BITMAPINFOHEADER(40) + 自底向上 BGRA 像素 + AND 掩码(未压缩图标需要)。
func encodeBMP(img *image.RGBA, s int) []byte {
	bw, bh := img.Bounds().Dx(), img.Bounds().Dy()

	// 每行 4 字节、按 4 字节对齐（32bpp 下天然对齐）
	rowBytes := uint32(bw) * 4
	imageBytes := rowBytes * uint32(bh)
	maskRowBytes := ((uint32(bw) + 31) / 32) * 4
	maskBytes := maskRowBytes * uint32(bh)
	total := uint32(40) + imageBytes + maskBytes

	out := make([]byte, 0, total)
	le := binary.LittleEndian

	// BITMAPINFOHEADER
	var h [40]byte
	le.PutUint32(h[0:4], 40)                 // biSize
	le.PutUint32(h[4:8], uint32(bw))         // biWidth
	le.PutUint32(h[8:12], uint32(2*bh))      // biHeight: 颜色+掩码总高度
	le.PutUint16(h[12:14], 1)                // biPlanes
	le.PutUint16(h[14:16], 32)               // biBitCount
	le.PutUint32(h[16:20], 0)                // biCompression = BI_RGB
	le.PutUint32(h[20:24], imageBytes)       // biSizeImage
	le.PutUint32(h[24:28], 2835)             // biXPelsPerMeter (~72dpi)
	le.PutUint32(h[28:32], 2835)             // biYPelsPerMeter
	le.PutUint32(h[32:36], 0)                // biClrUsed
	le.PutUint32(h[36:40], 0)                // biClrImportant
	out = append(out, h[:]...)

	// 颜色平面：自底向上，BGRA
	for y := bh - 1; y >= 0; y-- {
		for x := 0; x < bw; x++ {
			p := img.RGBAAt(x, y)
			a := p.A
			out = append(out, p.B, p.G, p.R, a)
		}
	}

	// AND 掩码：1 = 透明。ICO 掩码行自顶向下，行距按 4 字节对齐。
	for y := 0; y < bh; y++ {
		row := make([]byte, maskRowBytes)
		for x := 0; x < bw; x++ {
			a := img.RGBAAt(x, y).A
			if a == 0 {
				row[x/8] |= 1 << uint(7-(x%8))
			}
		}
		out = append(out, row...)
	}
	return out
}

// renderGlyph 在 ss 倍分辨率上绘制，再降采样到 s x s 的 RGBA。
func renderGlyph(s, r, g, b int) *image.RGBA {
	ss := supersample
	S := s * ss
	img := image.NewRGBA(image.Rect(0, 0, S, S))

	draw := func(drawFn func(*image.RGBA)) {
		drawFn(img)
	}

	// 坐标系里 y 向下为正。几何按 1.0 = 1 像素(S) 的比例定义，再乘 S。
	// 入线：左下 -> 分叉点；三条出线：分叉点 -> 右侧三个端点。
	// 分叉点在 (0.22, 0.5)，入口在 (0.10, 0.88)，出口在 x=0.86 的 y=0.16/0.5/0.84。
	scale := float64(S)
	strokeW := 0.115 * scale // 笔画宽度（4x 渲染下约 4.6px，降采样后 16px 约 1.15px）
	dotR := 0.075 * scale

	fx, fy := 0.22*scale, 0.50*scale   // 分叉点
	ex, ey := 0.10*scale, 0.88*scale   // 入口端点
	tx := 0.86 * scale                  // 出口端点 x
	exitYs := []float64{0.16, 0.50, 0.84}

	type seg struct{ x1, y1, x2, y2 float64 }
	type dot struct{ x, y float64 }

	segments := []seg{{ex, ey, fx, fy}}
	for _, ty := range exitYs {
		segments = append(segments, seg{fx, fy, tx, ty * scale})
	}

	dots := []dot{{ex, ey}}
	for _, ty := range exitYs {
		dots = append(dots, dot{tx, ty * scale})
	}

	for _, s := range segments {
		draw(func(im *image.RGBA) { strokeLine(im, s.x1, s.y1, s.x2, s.y2, strokeW, r, g, b) })
	}
	for _, d := range dots {
		draw(func(im *image.RGBA) { fillCircle(im, d.x, d.y, dotR, r, g, b) })
	}

	return downsample(img, ss, s)
}

// strokeLine 在超采样缓冲上画一段圆头直线：逐像素测点到线段距离。
func strokeLine(img *image.RGBA, x1, y1, x2, y2, w float64, r, g, b int) {
	half := w / 2
	bd := img.Bounds()
	// 包围盒：线段外接框再向外扩 half（半笔画宽）。
	// 注意水平/垂直线段某一维跨度为 0，必须用扩而不是缩，否则区间倒置、
	// 循环体一次都不执行——这正是中段水平线漏画的成因。
	minx := int(mathFloor(minf(x1, x2)-half))
	maxx := int(mathFloor(maxf(x1, x2)+half))
	miny := int(mathFloor(minf(y1, y2)-half))
	maxy := int(mathFloor(maxf(y1, y2)+half))
	if minx < bd.Min.X {
		minx = bd.Min.X
	}
	if miny < bd.Min.Y {
		miny = bd.Min.Y
	}
	if maxx > bd.Max.X {
		maxx = bd.Max.X
	}
	if maxy > bd.Max.Y {
		maxy = bd.Max.Y
	}
	for y := miny; y < maxy; y++ {
		for x := minx; x < maxx; x++ {
			d := distToSegment(float64(x), float64(y), x1, y1, x2, y2)
			if d <= half {
				blendPixel(img, x, y, coverage(half, 1.0, d), r, g, b)
			}
		}
	}
}

// fillCircle 画实心圆，边缘一像素内做覆盖渐变。
func fillCircle(img *image.RGBA, cx, cy, radius float64, r, g, b int) {
	bd := img.Bounds()
	minx, maxx := int(cx-radius)+1, int(cx+radius)-1
	miny, maxy := int(cy-radius)+1, int(cy+radius)-1
	for y := miny; y <= maxy; y++ {
		if y < bd.Min.Y || y >= bd.Max.Y {
			continue
		}
		for x := minx; x <= maxx; x++ {
			if x < bd.Min.X || x >= bd.Max.X {
				continue
			}
			d := mathSqrt((float64(x)-cx)*(float64(x)-cx) + (float64(y)-cy)*(float64(y)-cy))
			if d <= radius {
				blendPixel(img, x, y, coverage(0, 1.0, d-radius), r, g, b)
			}
		}
	}
}

// coverage 把"距离边缘的有符号量"映射到 0..1 覆盖率。edge=0 处全填充，
// 向外 1px 内线性衰减。
func coverage(edge, span, d float64) float64 {
	if span <= 0 {
		return 0
	}
	// 在边界内侧 0..1 之间渐变
	if d <= edge {
		return 1
	}
	if d >= edge+span {
		return 0
	}
	return (edge + span - d) / span
}

// blendPixel 用覆盖率做 alpha 混合（目标初始为全透明黑）。
func blendPixel(img *image.RGBA, x, y int, cov float64, r, g, b int) {
	if cov <= 0 {
		return
	}
	if cov > 1 {
		cov = 1
	}
	p := img.Pix[(y*img.Stride)+x*4 : (y*img.Stride)+(x+1)*4]
	p[0] = uint8(float64(r) * cov)
	p[1] = uint8(float64(g) * cov)
	p[2] = uint8(float64(b) * cov)
	p[3] = uint8(255 * cov)
}

// downsample 把 ss 倍缓冲按 ss x ss 块平均降到 s x s。
func downsample(src *image.RGBA, ss, s int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, s, s))
	for oy := 0; oy < s; oy++ {
		for ox := 0; ox < s; ox++ {
			var cr, cg, cb, ca float64
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					p := src.RGBAAt(ox*ss+dx, oy*ss+dy)
					cr += float64(p.R)
					cg += float64(p.G)
					cb += float64(p.B)
					ca += float64(p.A)
				}
			}
			n := float64(ss * ss)
			c := color.RGBA{uint8(cr / n), uint8(cg / n), uint8(cb / n), uint8(ca / n)}
			out.SetRGBA(ox, oy, c)
		}
	}
	return out
}

// icoBytes 在内存里拼出完整 ICO，用于校验与报体积。
func icoBytes(dirs []iconDir, blobs [][]byte) []byte {
	var buf bytes.Buffer
	le := binary.LittleEndian
	buf.Write([]byte{0, 0})                       // reserved
	binary.Write(&buf, le, uint16(1))             // type: ICO
	binary.Write(&buf, le, uint16(len(dirs)))     // count
	headerSize := uint32(6)
	off := headerSize + uint32(16*len(dirs))
	for _, d := range dirs {
		buf.WriteByte(d.wWidth)
		buf.WriteByte(d.hHeight)
		buf.WriteByte(d.colorCount)
		buf.WriteByte(d.reserved)
		binary.Write(&buf, le, d.planes)
		binary.Write(&buf, le, d.bitCount)
		binary.Write(&buf, le, d.bytesInRes)
		binary.Write(&buf, le, d.imageOffset)
		_ = off
	}
	for _, blob := range blobs {
		buf.Write(blob)
	}
	return buf.Bytes()
}

func writeICO(path string, dirs []iconDir, blobs [][]byte) error {
	var buf bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&buf, le, uint16(0))
	binary.Write(&buf, le, uint16(1))
	binary.Write(&buf, le, uint16(len(dirs)))
	for _, d := range dirs {
		buf.WriteByte(d.wWidth)
		buf.WriteByte(d.hHeight)
		buf.WriteByte(d.colorCount)
		buf.WriteByte(d.reserved)
		binary.Write(&buf, le, d.planes)
		binary.Write(&buf, le, d.bitCount)
		binary.Write(&buf, le, d.bytesInRes)
		binary.Write(&buf, le, d.imageOffset)
	}
	for _, blob := range blobs {
		buf.Write(blob)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func writePreview(path string, img *image.RGBA) error {
	out := filepath.Join(repoRoot(), ".superpowers", "sdd", path)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// repoRoot 返回仓库根目录。`go run ./tools/genicon` 把工作目录设为仓库根，
// 所以这里直接返回当前目录；若有人从 tools/genicon 内直接运行 go run .，
// 则退回上一级的上一级。用 go.mod 作为锚点判定，避免两种情况都出错。
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// verifyICO 解析刚写的 ICO 结构并逐字段自检。
func verifyICO(dirs []iconDir, blobs [][]byte) {
	ico := icoBytes(dirs, blobs)
	ok := true

	read := func(off int, n int) []byte {
		return ico[off : off+n]
	}

	if len(read(0, 2)) != 2 || binary.LittleEndian.Uint16(read(0, 2)) != 0 {
		fmt.Println("verify: FAIL reserved != 0")
		ok = false
	}
	if binary.LittleEndian.Uint16(read(2, 2)) != 1 {
		fmt.Println("verify: FAIL type != 1 (ICO)")
		ok = false
	}
	count := int(binary.LittleEndian.Uint16(read(4, 2)))
	fmt.Printf("verify: header ok (type=1, count=%d, total=%d bytes)\n", count, len(ico))
	if count != len(dirs) {
		fmt.Println("verify: FAIL count mismatch")
		ok = false
	}

	expected := []int{16, 32, 48, 256}
	for i, d := range dirs {
		e := 6 + i*16
		off := int(binary.LittleEndian.Uint32(read(e+12, 4)))
		size := int(binary.LittleEndian.Uint32(read(e+8, 4)))

		w := int(d.wWidth)
		if w == 0 {
			w = 256
		}
		h := int(d.hHeight)
		if h == 0 {
			h = 256
		}

		prob := []string{}
		if w != expected[i] {
			prob = append(prob, fmt.Sprintf("width %d != %d", w, expected[i]))
		}
		if h != w {
			prob = append(prob, fmt.Sprintf("height %d != width %d", h, w))
		}
		if off+size > len(ico) {
			prob = append(prob, fmt.Sprintf("offset+size %d+%d exceeds file %d", off, size, len(ico)))
		}
		if len(blobs[i]) != size {
			prob = append(prob, fmt.Sprintf("declared %d != actual blob %d", size, len(blobs[i])))
		}
		if !bytes.Equal(ico[off:off+size], blobs[i]) {
			prob = append(prob, "embedded blob differs from rendered blob")
		}

		if w >= 256 {
			// PNG 路径：检查 PNG 魔数
			if !bytes.HasPrefix(ico[off:off+8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
				prob = append(prob, "missing PNG magic")
			}
			if d.wWidth != 0 || d.hHeight != 0 {
				prob = append(prob, "256 entry must encode width/height as 0")
			}
			img, perr := png.Decode(bytes.NewReader(ico[off : off+size]))
			if perr != nil {
				prob = append(prob, "PNG undecodable: "+perr.Error())
			} else if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
				prob = append(prob, fmt.Sprintf("PNG dim %dx%d != 256", img.Bounds().Dx(), img.Bounds().Dy()))
			}
		} else {
			if d.bitCount != 32 {
				prob = append(prob, fmt.Sprintf("bitCount %d != 32", d.bitCount))
			}
			if d.planes != 1 {
				prob = append(prob, fmt.Sprintf("planes %d != 1", d.planes))
			}
			if d.colorCount != 0 {
				prob = append(prob, "colorCount != 0")
			}
			dib := ico[off : off+size]
			if len(dib) < 40 {
				prob = append(prob, "DIB header truncated")
			} else {
				biSize := binary.LittleEndian.Uint32(dib[0:4])
				biW := int(binary.LittleEndian.Uint32(dib[4:8]))
				biH := int(binary.LittleEndian.Uint32(dib[8:12]))
				biCompression := binary.LittleEndian.Uint32(dib[16:20])
				if biSize != 40 {
					prob = append(prob, fmt.Sprintf("biSize %d != 40", biSize))
				}
				if biW != w {
					prob = append(prob, fmt.Sprintf("biWidth %d != %d", biW, w))
				}
				if biH != 2*w {
					prob = append(prob, fmt.Sprintf("biHeight %d != 2*%d (color+mask)", biH, w))
				}
				if biCompression != 0 {
					prob = append(prob, "biCompression != BI_RGB")
				}
				rowBytes := uint32(biW) * 4
				maskRow := ((uint32(biW) + 31) / 32) * 4
				wantTotal := uint32(40) + rowBytes*uint32(biW) + maskRow*uint32(biW)
				if uint32(len(dib)) != wantTotal {
					prob = append(prob, fmt.Sprintf("DIB total %d != computed %d", len(dib), wantTotal))
				}
			}
		}

		if len(prob) == 0 {
			fmt.Printf("verify: entry %d ok: %dx%d wWidth=%d hHeight=%d bitCount=%d planes=%d size=%d offset=%d\n",
				i, w, h, d.wWidth, d.hHeight, d.bitCount, d.planes, size, off)
		} else {
			ok = false
			for _, p := range prob {
				fmt.Printf("verify: entry %d FAIL: %s\n", i, p)
			}
		}
	}

	if ok {
		fmt.Println("verify: ALL CHECKS PASSED")
	} else {
		fmt.Println("verify: FAILED")
		os.Exit(1)
	}
}

// ---- 小工具 ----

func distToSegment(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	if dx == 0 && dy == 0 {
		return mathSqrt((px-x1)*(px-x1) + (py-y1)*(py-y1))
	}
	t := ((px-x1)*dx + (py-y1)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := x1+t*dx, y1+t*dy
	return mathSqrt((px-cx)*(px-cx) + (py-cy)*(py-cy))
}

// mathSqrt 保留薄封装，便于将来换成快速近似（当前直接委托 math.Sqrt）。
func mathSqrt(v float64) float64 { return math.Sqrt(v) }

// mathFloor 保留薄封装，同 mathSqrt。
func mathFloor(v float64) float64 { return math.Floor(v) }

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
