package li

import (
	"fmt"
	"path"
)

type evRenderStatus struct{}

var EvRenderStatus = new(evRenderStatus)

type AddStatusLine func(...dyn)

func Status(
	scope Scope,
	box Box,
	cur CurrentView,
	getStyle GetStyle,
	style Style,
	curGroup CurrentViewGroup,
	groups ViewGroups,
	trigger Trigger,
) (
	ret Element,
) {

	focusing := cur()
	style = darkerOrLighterStyle(style, 15)
	hlStyle := getStyle("Highlight")
	fg, _, _ := hlStyle.Decompose()
	hlStyle = style.Foreground(fg)

	lineBox := Box{
		Top:    box.Top,
		Left:   box.Left,
		Right:  box.Right,
		Bottom: box.Top + 1,
	}
	var subs []Element

	addTextLine := func(specs ...any) {
		specs = append(specs, lineBox)
		subs = append(subs, Text(specs...))
		lineBox.Top++
		lineBox.Bottom++
	}

	trigger(scope.Sub(
		func() AddStatusLine {
			return addTextLine
		},
		func() []Style {
			return []Style{style, hlStyle}
		},
	), EvRenderStatus)

	group := curGroup()
	groupIndex := func() int {
		for i, g := range groups {
			if g == group {
				return i
			}
		}
		return 0
	}()

	// views
	views := group.GetViews(scope)
	if len(views) > 0 {
		addTextLine("")
		addTextLine(
			fmt.Sprintf("group %d / %d", groupIndex+1, len(groups)),
			Bold(true), AlignRight, Padding(0, 2, 0, 0))
		box := Box{
			Top:    lineBox.Top,
			Left:   box.Left,
			Right:  box.Right,
			Bottom: box.Bottom,
		}
		focusLine := func() int {
			for i, view := range views {
				if view == focusing {
					return i
				}
			}
			return 0
		}()
		subs = append(subs, ElementWith(
			VerticalScroll(
				ElementFrom(func(
					box Box,
				) (ret []Element) {
					for i, view := range views {
						name := path.Base(view.Buffer.Path)
						s := style
						if view == focusing {
							s = hlStyle
						}
						if view.Buffer.LastSyncFileInfo == view.GetMoment().FileInfo {
							s = s.Underline(false)
						} else {
							s = s.Underline(true)
						}
						ret = append(ret, Text(
							Box{
								Top:    box.Top + i,
								Left:   box.Left,
								Right:  box.Right,
								Bottom: box.Top + i + 1,
							},
							name,
							s,
							AlignRight,
							Padding(0, 2, 0, 0),
						))
					}
					return
				}),
				focusLine,
			),
			func() Box {
				return box
			},
		))
	}

	return Rect(
		style,
		box,
		Fill(true),
		subs,
	)

}
