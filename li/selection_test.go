package li

import "testing"

func TestSelection(t *testing.T) {
	withHelloEditor(t, func(
		scope Scope,
		view *View,
		move MoveCursor,
	) {

		scope.Call(ToggleSelection)
		move(Move{RelLine: 1})
		r := view.selectedRange()
		eq(t,
			r.Begin.Line, 0,
			r.Begin.Cell, 0,
			r.End.Line, 1,
			r.End.Cell, 1,
		)
		scope.Call(ToggleSelection)
		r = view.selectedRange()
		eq(t,
			r == nil, true,
		)

		scope.Call(ToggleSelection)
		move(Move{RelLine: -1})
		r = view.selectedRange()
		eq(t,
			r.Begin.Line, 0,
			r.Begin.Cell, 0,
			r.End.Line, 1,
			r.End.Cell, 0,
		)

	})
}

func TestSelection2(t *testing.T) {
	withEditor(func(
		scope Scope,
	) {

		scope.Call(ToggleSelection)

	})
}
