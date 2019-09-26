package li

import (
	"C"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unsafe"

	"github.com/reusee/li/treesitter"
)
import (
	"bytes"
	"unicode/utf16"
	"unicode/utf8"
)

type MomentID int64

type Moment struct {
	T0 time.Time

	ID       MomentID
	Previous *Moment
	Change   Change
	lines    []*Line

	FileInfo FileInfo

	initContentOnce        sync.Once
	content                string
	initLowerContentOnce   sync.Once
	lowerContent           string
	initCStringContentOnce sync.Once
	cstringContent         *C.char
	initBytesOnce          sync.Once
	bytes                  []byte

	initParserOnce sync.Once
	parser         *treesitter.Parser
	syntaxAttrs    sync.Map

	finalizeFuncs sync.Map
}

func NewMoment(prev *Moment) *Moment {
	m := &Moment{
		T0:       time.Now(),
		ID:       MomentID(atomic.AddInt64(&nextMomentID, 1)),
		Previous: prev,
	}
	runtime.SetFinalizer(m, func(m *Moment) {
		m.finalizeFuncs.Range(func(_, v any) bool {
			v.(func())()
			return true
		})
	})
	return m
}

func (m *Moment) GetLine(scope Scope, i int) *Line {
	if i < 0 {
		return nil
	}
	if i >= m.NumLines() {
		return nil
	}
	line := m.lines[i]
	line.init(scope)
	return line
}

func (m *Moment) GetContent() string {
	m.initContentOnce.Do(func() {
		var b strings.Builder
		for _, line := range m.lines {
			b.WriteString(line.content)
		}
		m.content = b.String()
	})
	return m.content
}

func (m *Moment) GetLowerContent() string {
	m.initLowerContentOnce.Do(func() {
		content := m.GetContent()
		m.lowerContent = strings.ToLower(content)
	})
	return m.lowerContent
}

func (m *Moment) GetCStringContent() *C.char {
	m.initCStringContentOnce.Do(func() {
		content := C.CString(m.GetContent())
		m.finalizeFuncs.Store(rand.Int63(), func() {
			cfree(unsafe.Pointer(m.cstringContent))
		})
		m.cstringContent = content
	})
	return m.cstringContent
}

func (m *Moment) GetBytes() []byte {
	m.initBytesOnce.Do(func() {
		var b bytes.Buffer
		for _, line := range m.lines {
			b.WriteString(line.content)
		}
		m.bytes = b.Bytes()
	})
	return m.bytes
}

func (m *Moment) GetParser(scope Scope) *treesitter.Parser {
	var buffer *Buffer
	var linked LinkedOne
	scope.Assign(&linked)
	linked(m, &buffer)
	if buffer.language == LanguageUnknown {
		return nil
	}
	m.initParserOnce.Do(func() {
		//TODO utilize tree-sitter incremental parsing
		if fn, ok := languageParsers[buffer.language]; ok {
			m.parser = fn(m)
		}
		m.finalizeFuncs.Store(rand.Int63(), func() {
			m.parser.Close()
		})
	})
	return m.parser
}

func (m *Moment) GetSyntaxAttr(scope Scope, lineNum int, runeOffset int) string {
	key := Position{
		Line: lineNum,
		Cell: runeOffset,
	}
	if v, ok := m.syntaxAttrs.Load(key); ok {
		return v.(string)
	}
	parser := m.GetParser(scope)
	if parser == nil {
		return ""
	}
	node := parser.NodeAt(treesitter.Point(lineNum, runeOffset))
	nodeType := treesitter.NodeType(node)
	attr := nodeType
	m.syntaxAttrs.Store(key, attr)
	return attr
}

func (m *Moment) NumLines() int {
	return len(m.lines)
}

func (m *Moment) ByteOffsetToPosition(scope Scope, offset int) (pos Position) {
	for i, line := range m.lines {
		if offset < len(line.content) {
			line.init(scope)
			for _, cell := range line.Cells {
				if offset < cell.Len {
					pos.Cell = cell.RuneOffset
					return
				}
				offset -= cell.Len
			}
		} else {
			offset -= len(line.content)
			pos.Line = i + 1
		}
	}
	return
}

type Line struct {
	Cells                 []Cell
	Runes                 []rune
	DisplayWidth          int
	AllSpace              bool
	NonSpaceDisplayOffset *int

	content  string
	initOnce *sync.Once
	config   *BufferConfig
}

func (l *Line) init(scope Scope) {
	l.initOnce.Do(func() {
		var cells []Cell
		allSpace := true
		displayOffset := 0
		utf16ByteOffset := 0
		byteOffset := 0
		l.Runes = []rune(l.content)
		var nonSpaceOffset *int
		for i, r := range l.Runes {
			width := runeWidth(r)
			var displayWidth int
			if r == '\t' && l.config.ExpandTabs {
				displayWidth = l.config.TabWidth
			} else {
				displayWidth = width
			}
			runeLen := utf8.RuneLen(r)
			cell := Cell{
				Rune:          r,
				Len:           runeLen,
				Width:         width,
				DisplayWidth:  displayWidth,
				DisplayOffset: displayOffset,
				RuneOffset:    i,
				ByteOffset:    byteOffset,
				UTF16Offset:   utf16ByteOffset,
			}
			cells = append(cells, cell)
			l.DisplayWidth += cell.DisplayWidth
			if !unicode.IsSpace(r) {
				allSpace = false
				if nonSpaceOffset == nil {
					offset := displayOffset
					nonSpaceOffset = &offset
				}
			}
			displayOffset += displayWidth
			utf16ByteOffset += len(utf16.Encode([]rune{r})) * 2
			byteOffset += runeLen
		}
		l.NonSpaceDisplayOffset = nonSpaceOffset
		l.Cells = cells
		l.AllSpace = allSpace

		var trigger Trigger
		scope.Assign(&trigger)
		trigger(scope.Sub(func() *Line {
			return l
		}), EvLineInitialized)
	})
}

type evLineInitialized struct{}

var EvLineInitialized = new(evLineInitialized)

type Cell struct {
	Rune          rune
	Len           int // number of bytes in utf8 encoding
	Width         int // visual width without padding
	DisplayWidth  int // visual width with padding
	DisplayOffset int // visual column offset with padding in line
	RuneOffset    int // rune offset in line
	ByteOffset    int // utf8 byte offset in line
	UTF16Offset   int // byte offset in utf16 encoding in line
}

var nextMomentID int64

func NewMomentFromFile(
	path string,
	scope Scope,
) (
	moment *Moment,
	linebreak Linebreak,
	err error,
) {
	defer he(&err)

	// read
	contentBytes, err := ioutil.ReadFile(path)
	ce(err, "read %s", path)

	scope.Sub(func() []byte {
		return contentBytes
	}).Call(NewMomentFromBytes, &moment, &linebreak)

	info, err := getFileInfo(path)
	ce(err)
	moment.FileInfo = info

	return
}

func NewMomentFromBytes(
	bs []byte,
	scope Scope,
	config BufferConfig,
	initProcs LineInitProcs,
) (
	moment *Moment,
	linebreak Linebreak,
) {

	linebreak = "\n" // default

	content := string(bs)

	// split
	lineContents := splitLines(content)
	n := 0
	for i, lineContent := range lineContents {
		noCR := strings.TrimSuffix(lineContent, "\r")
		if len(noCR) != len(lineContent) {
			lineContents[i] = noCR
			n++
		}
	}
	if float64(n)/float64(len(lineContents)) > 0.4 {
		linebreak = "\r\n"
	}

	// lines
	var lines []*Line
	for _, content := range lineContents {
		line := &Line{
			content:  content,
			initOnce: new(sync.Once),
			config:   &config,
		}
		lines = append(lines, line)
	}
	initProcs <- lines

	moment = NewMoment(nil)
	moment.lines = lines

	return
}

func NewMomentsFromPath(
	path string,
	scope Scope,
) (
	moments []*Moment,
	linebreaks []Linebreak,
	paths []string,
	err error,
) {

	stat, err := os.Stat(path)
	if err != nil {
		return
	}

	if stat.IsDir() {
		var f *os.File
		f, err = os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		for {
			infos, err := f.Readdir(256)
			for _, info := range infos {
				if info.IsDir() {
					continue
				}
				name := info.Name()
				p := filepath.Join(path, name)
				var moment *Moment
				var linebreak Linebreak
				scope.Sub(func() string {
					return p
				}).Call(NewMomentFromFile, &moment, &linebreak, &err)
				if err != nil {
					continue
				}
				moments = append(moments, moment)
				linebreaks = append(linebreaks, linebreak)
				paths = append(paths, p)
			}
			if err == nil {
				break
			}
		}

	} else {
		var moment *Moment
		var linebreak Linebreak
		scope.Sub(func() string {
			return path
		}).Call(NewMomentFromFile, &moment, &linebreak, &err)
		if err != nil {
			return
		}
		moments = append(moments, moment)
		linebreaks = append(linebreaks, linebreak)
		paths = append(paths, path)
	}

	return
}

func splitLines(s string) (ret []string) {
	if s == "" {
		ret = append(ret, "")
		return
	}
	for len(s) > 0 {
		i := strings.Index(s, "\n")
		if i == -1 {
			ret = append(ret, s)
			return
		}
		ret = append(ret, s[:i+1])
		s = s[i+1:]
	}
	return
}

type LineInitProcs chan []*Line

func (_ Provide) LineInitProcs(
	scope Scope,
) LineInitProcs {
	c := make(chan []*Line, 512)
	for i := 0; i < numCPU; i++ {
		go func() {
			for {
				lines := <-c
				for i := len(lines) - 1; i >= 0; i-- {
					lines[i].init(scope)
				}
			}
		}()
	}
	return c
}
