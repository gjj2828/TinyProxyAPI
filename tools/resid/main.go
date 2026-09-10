// tools/resid 解析一个 Windows PE 可执行文件的资源目录，打印
// RT_GROUP_ICON / RT_ICON / RT_MANIFEST 等类型下的资源 ID。
//
// 用途：确认 rsrc 给图标分配的 GROUP_ICON 资源 ID 到底是多少。规格要求
// 该常量必须来自实测，不得凭推测填写；本工具就是这条"实测"链路的执行端。
//
// 用法：go run ./tools/resid path/to/prog.exe
//
// 保留在仓库里的理由：图标是构建产物，将来任何人重新生成 icon.ico /
// rsrc.syso 后，都可以用同一套代码重新确认 ID，而不是再去翻 rsrc 源码
// 或凭记忆。它是纯标准库的一次性诊断脚本，不进入主程序依赖链。
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// 常见资源类型号。
const (
	rtBitmap      = 2
	rtIcon        = 3
	rtGroupCursor = 12
	rtGroupIcon   = 14
	rtManifest    = 24
)

var typeNames = map[uint32]string{
	1:  "RT_CURSOR",
	rtBitmap:      "RT_BITMAP",
	rtIcon:        "RT_ICON",
	rtGroupCursor: "RT_GROUP_CURSOR",
	rtGroupIcon:   "RT_GROUP_ICON",
	rtManifest:    "RT_MANIFEST",
}

var langNames = map[uint32]string{
	0x0409: "en-US",
	0x0804: "zh-CN",
	0x0400: "LangNeutral",
}

// ---- 小端读取助手（避免到处写切片运算） ----

func u16(data []byte, off uint32) uint16 { return binary.LittleEndian.Uint16(data[off : off+2]) }
func u32(data []byte, off uint32) uint32 { return binary.LittleEndian.Uint32(data[off : off+4]) }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: resid <exe>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s (%d bytes)\n", os.Args[1], len(data))

	secBase, dirOff, found := locateResourceDir(data)
	if !found {
		fmt.Fprintln(os.Stderr, "no resource directory found")
		os.Exit(1)
	}
	fmt.Printf("resource directory: section base 0x%x, IMAGE_RESOURCE_DIRECTORY at file offset 0x%x\n\n",
		secBase, dirOff)

	root := dirOff
	fmt.Printf("IMAGE_RESOURCE_DIRECTORY @0x%x: %s\n", root,
		describeDir(data, root))
	walk(data, secBase, root, 0, "", "")
}

// locateResourceDir 返回 (.rsrc 节的文件基址, IMAGE_RESOURCE_DIRECTORY 的文件偏移)。
func locateResourceDir(data []byte) (uint32, uint32, bool) {
	if len(data) < 64 || string(data[:2]) != "MZ" {
		return 0, 0, false
	}
	peOff := binary.LittleEndian.Uint32(data[0x3C:0x40])
	if string(data[peOff:peOff+4]) != "PE\x00\x00" {
		return 0, 0, false
	}
	fileHdr := peOff + 4 // IMAGE_FILE_HEADER
	nsec := u16(data, fileHdr+2)
	optSize := u16(data, fileHdr+16)

	opt := fileHdr + 20
	magic := u16(data, opt)
	// PE32+ 的可选头更长，数据目录起始偏移不同
	var ddirRel uint32
	switch magic {
	case 0x020B: // PE32+
		ddirRel = 112
	case 0x010B: // PE32
		ddirRel = 96
	default:
		return 0, 0, false
	}

	ddir := opt + ddirRel
	// 数据目录顺序：0=导出表, 1=导入表, 2=资源表。资源目录是索引 2。
	// （注意不是 1——索引 1 是导入表，拿错会把 .idata 当成资源树解析。）
	res := ddir + 2*8
	va := u32(data, res+0)
	sz := u32(data, res+4)
	if va == 0 || sz == 0 {
		return 0, 0, false
	}

	secTbl := opt + uint32(optSize)
	for i := 0; i < int(nsec); i++ {
		s := secTbl + uint32(i)*40
		secVA := u32(data, s+12)
		secVSz := u32(data, s+8)
		secRaw := u32(data, s+20)
		if va >= secVA && va < secVA+secVSz {
			return secRaw, secRaw + (va - secVA), true
		}
	}
	return 0, 0, false
}

// describeDir 打印一个目录的概要（子目录数、数据叶子数）。
func describeDir(data []byte, off uint32) string {
	if off+16 > uint32(len(data)) {
		return "??"
	}
	nc := u16(data, off+12)
	ic := u16(data, off+14)
	return fmt.Sprintf("chars=0x%x ts=0x%x v%02d.%02d named=%d id'd=%d",
		u32(data, off),
		u32(data, off+4),
		u16(data, off+8), u16(data, off+10),
		nc, ic)
}

// walk 递归遍历。lvl 0=类型, 1=id/名称, 2=语言, 3=数据叶子。
func walk(data []byte, secBase, dirOff uint32, lvl int, typeName, idName string) {
	rd := dirOff
	if rd+16 > uint32(len(data)) {
		return
	}
	nameCount := u16(data, rd+12)
	idCount := u16(data, rd+14)
	entries := uint32(nameCount) + uint32(idCount)

	for i := uint32(0); i < entries; i++ {
		e := rd + 16 + i*8
		if e+8 > uint32(len(data)) {
			return
		}
		key := u32(data, e+0)
		dir := u32(data, e+4)

		switch lvl {
		case 0:
			typeName = typeLabel(key)
		case 1:
			idName = idLabel(data, secBase, key)
		case 2:
			fmt.Printf("lang %s\n", langLabel(key))
		}

		if dir&0x80000000 != 0 {
			walk(data, secBase, secBase+(dir&0x7FFFFFFF), lvl+1, typeName, idName)
			continue
		}
		// 叶子：IMAGE_RESOURCE_DATA_ENTRY
		d := secBase + (dir & 0x7FFFFFFF)
		if d+16 > uint32(len(data)) {
			continue
		}
		dataVA := u32(data, d+0)
		dataSz := u32(data, d+4)
		lang := langLabel(u32(data, e+0))
		fmt.Printf("resource %s / %s / %s  -> RVA 0x%x size %d bytes\n",
			typeName, idName, lang, dataVA, dataSz)
	}
}

func typeLabel(key uint32) string {
	if n, ok := typeNames[key]; ok {
		return n
	}
	return fmt.Sprintf("0x%x(%d)", key, key)
}

func idLabel(data []byte, secBase, key uint32) string {
	if key&0x80000000 != 0 {
		// 按名称索引
		p := secBase + (key & 0x7FFFFFFF)
		if p+2 <= uint32(len(data)) {
			n := u16(data, p)
			if p+2+uint32(2*int(n)) <= uint32(len(data)) {
				var b []byte
				raw := data[p+2 : p+2+uint32(2*int(n))]
				for i := 0; i+1 < len(raw); i += 2 {
					b = append(b, raw[i])
				}
				return fmt.Sprintf("\"%s\"", string(b))
			}
		}
		return "<name>"
	}
	return fmt.Sprintf("%d (0x%x)", key, key)
}

func langLabel(key uint32) string {
	if n, ok := langNames[key]; ok {
		return n
	}
	return fmt.Sprintf("0x%04x", key)
}
