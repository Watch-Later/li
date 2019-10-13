package li

func (_ Command) InsertNewline() (spec CommandSpec) {
	spec.Desc = "insert newline at cursor"
	spec.Func = func(
		scope Scope,
		cur CurrentView,
	) {
		view := cur()
		if view == nil {
			return
		}
		indent := getAdjacentIndent(scope, view, view.CursorLine, view.CursorLine+1)
		scope.Sub(func() (PositionFunc, string) {
			return PosCursor, "\n" + indent
		}).Call(InsertAtPositionFunc)
	}
	return
}

func (_ Command) InsertTab() (spec CommandSpec) {
	spec.Desc = "insert tab at cursor"
	spec.Func = func(
		scope Scope,
	) {
		scope.Sub(func() (PositionFunc, string) {
			return PosCursor, "\t"
		}).Call(InsertAtPositionFunc)
	}
	return
}

func (_ Command) Append() (spec CommandSpec) {
	spec.Desc = "start append at current cursor"
	spec.Func = func(scope Scope) {
		scope.Sub(&Move{RelRune: 1}).Call(MoveCursor)
		scope.Call(EnableEditMode)
	}
	return
}

func (_ Command) DeletePrevRune() (spec CommandSpec) {
	spec.Desc = "delete previous rune at cursor"
	spec.Func = DeletePrevRune
	return
}

func (_ Command) DeleteRune() (spec CommandSpec) {
	spec.Desc = "delete one rune at cursor"
	spec.Func = DeleteRune
	return
}

func (_ Command) Delete() (spec CommandSpec) {
	spec.Desc = "delete selected or text object"
	spec.Func = Delete
	return
}

func (_ Command) Change() (spec CommandSpec) {
	spec.Desc = "change selected or text object"
	spec.Func = ChangeText
	return
}

func (_ Command) EditNewLineBelow() (spec CommandSpec) {
	spec.Desc = "insert new line below the current line and enable edit mode"
	spec.Func = func(
		scope Scope,
		cur CurrentView,
	) {
		view := cur()
		if view == nil {
			return
		}
		indent := getAdjacentIndent(scope, view, view.CursorLine, view.CursorLine+1)
		scope.Sub(func() (PositionFunc, string, *View) {
			return PosLineEnd, "\n" + indent, view
		}).Call(InsertAtPositionFunc)
		scope.Call(LineEnd)
		scope.Call(EnableEditMode)
	}
	return
}

func (_ Command) EditNewLineAbove() (spec CommandSpec) {
	spec.Desc = "insert new line above the current line and enable edit mode"
	spec.Func = func(
		scope Scope,
		cur CurrentView,
	) {
		view := cur()
		if view == nil {
			return
		}
		indent := getAdjacentIndent(scope, view, view.CursorLine-1, view.CursorLine)
		scope.Sub(func() (PositionFunc, string, *View) {
			return PosLineBegin, indent + "\n", view
		}).Call(InsertAtPositionFunc)
		scope.Sub(&Move{RelLine: -1}).Call(MoveCursor)
		scope.Call(LineEnd)
		scope.Call(EnableEditMode)
	}
	return
}

func getIndent(scope Scope, view *View, lineNum int) string {
	line := view.GetMoment().GetLine(scope, lineNum)
	if line == nil {
		return ""
	}
	if line.NonSpaceDisplayOffset == nil {
		return ""
	}
	var runes []rune
	for _, cell := range line.Cells {
		if cell.DisplayOffset >= *line.NonSpaceDisplayOffset {
			break
		}
		runes = append(runes, cell.Rune)
	}
	return string(runes)
}

func getAdjacentIndent(scope Scope, view *View, upwardLine int, downwardLine int) string {
	upwardIndent := 0
	var upwardRunes []rune
	for {
		line := view.GetMoment().GetLine(scope, upwardLine)
		if line == nil {
			break
		}
		if line.NonSpaceDisplayOffset == nil {
			upwardLine--
			continue
		}
		if *line.NonSpaceDisplayOffset > upwardIndent {
			upwardIndent = *line.NonSpaceDisplayOffset
			for _, cell := range line.Cells {
				if cell.DisplayOffset >= upwardIndent {
					break
				}
				upwardRunes = append(upwardRunes, cell.Rune)
			}
		}
		break
	}

	downwardIndent := 0
	var downwardRunes []rune
	for {
		line := view.GetMoment().GetLine(scope, downwardLine)
		if line == nil {
			break
		}
		if line.NonSpaceDisplayOffset == nil {
			downwardLine++
			continue
		}
		if *line.NonSpaceDisplayOffset > downwardIndent {
			downwardIndent = *line.NonSpaceDisplayOffset
			for _, cell := range line.Cells {
				if cell.DisplayOffset >= downwardIndent {
					break
				}
				downwardRunes = append(downwardRunes, cell.Rune)
			}
		}
		break
	}

	if upwardIndent > downwardIndent {
		return string(upwardRunes)
	}
	return string(downwardRunes)
}

func (_ Command) ChangeToWordEnd() (spec CommandSpec) {
	spec.Desc = "change text from current cursor position to end of word"
	spec.Func = ChangeToWordEnd
	return
}

func (_ Command) DeleteLine() (spec CommandSpec) {
	spec.Desc = "delete current line"
	spec.Func = DeleteLine
	return
}

func (_ Command) AppendAtLineEnd() (spec CommandSpec) {
	spec.Desc = "append at line end"
	spec.Func = func(scope Scope) {
		scope.Call(LineEnd)
		scope.Call(EnableEditMode)
	}
	return
}

func (_ Command) ChangeLine() (spec CommandSpec) {
	spec.Desc = "change current line"
	spec.Func = func(
		scope Scope,
		v CurrentView,
	) {
		view := v()
		if view == nil {
			return
		}
		indent := getIndent(scope, view, view.CursorLine)
		var begin, end Position
		scope.Call(PosLineBegin, &begin)
		scope.Call(PosLineEnd, &end)
		scope.Sub(func() (Range, string) {
			return Range{begin, end}, ""
		}).Call(ReplaceWithinRange)
		scope.Sub(func() (PositionFunc, string) {
			return PosCursor, indent
		}).Call(InsertAtPositionFunc)
		scope.Call(EnableEditMode)
	}
	return
}
