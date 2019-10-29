package li

type (
	LineHint struct {
		Moment *Moment
		Line   int
		Hints  []string
		mark   int
	}
	GetLineHints func() []LineHint
	AddLineHint  func(*Moment, int, []string)
)

type evCollectLineHints struct{}

var EvCollectLineHints = new(evCollectLineHints)

func (_ Provide) LineHints(
	on On,
) Init2 {

	// sorted
	var hints []LineHint

	changed := false
	mark := 42
	add := AddLineHint(func(
		moment *Moment,
		line int,
		strs []string,
	) {
		i, j := 0, len(hints)

		// binary search
		for i < j {
			h := int(uint(i+j) >> 1)
			hint := hints[h]
			if moment.ID > hint.Moment.ID {
				i = h + 1
			} else if moment.ID < hint.Moment.ID {
				j = h
			} else {
				if line > hint.Line {
					i = h + 1
				} else if line < hint.Line {
					j = h
				} else {
					// found, check strs
					same := true
					if len(strs) != len(hint.Hints) {
						same = false
					} else {
						for i, str := range strs {
							if str != hint.Hints[i] {
								same = false
								break
							}
						}
					}
					if !same {
						changed = true
						hints[h] = LineHint{
							Moment: moment,
							Line:   line,
							Hints:  strs,
							mark:   mark,
						}
					} else {
						hints[h].mark = mark
					}
					return
				}
			}
		}

		// not found, insert
		changed = true
		hints = append(
			hints[:i],
			append(
				[]LineHint{
					{
						Moment: moment,
						Line:   line,
						Hints:  strs,
						mark:   mark,
					},
				},
				hints[i:]...,
			)...,
		)
	})

	on(EvLoopBegin, func(
		trigger Trigger,
		scope Scope,
		cont ContinueMainLoop,
	) {
		changed = false
		mark++
		trigger(
			scope.Sub(&add),
			EvCollectLineHints,
		)
		// clear unmarked entries
		hs := hints[:0]
		for _, hint := range hints {
			if hint.mark == mark {
				hs = append(hs, hint)
			} else {
				changed = true
			}
		}
		if changed {
			hints = hs
			// mark re-render
			cont()
		}
	})

	return func() GetLineHints {
		return func() []LineHint {
			return hints
		}
	}

}