package li

import "testing"

func TestScopeValue(t *testing.T) {
	withEditor(func(
		scope Scope,
	) {

		type Elem int
		type Access func(...Elem) Elem

		provider := SeriesValue{
			Type:   Elem(0),
			Access: Access(nil),
		}.Provider()
		scope = scope.Sub(provider)
		scope.Call(func(
			access Access,
		) {
			access(42)
			if access() != 42 {
				t.Fatal()
			}
		})

		ok := false
		provider = SeriesValue{
			Type:   Elem(0),
			Access: Access(nil),
			OnLink: func(e Elem) {
				ok = true
			},
		}.Provider()
		scope = scope.Sub(provider)
		scope.Call(func(
			access Access,
		) {
			access(42)
			if access() != 42 {
				t.Fatal()
			}
			if !ok {
				t.Fatal()
			}
		})

		ok = false
		provider = SeriesValue{
			Type:   Elem(0),
			Access: Access(nil),
			OnLink: func(e Elem) {
				ok = true
			},
		}.Provider()
		scope = scope.Sub(provider)
		scope.Call(func(
			access Access,
		) {
			access()
			if ok {
				t.Fatal()
			}
		})

		ok = false
		provider = SeriesValue{
			Type:   Elem(0),
			Access: Access(nil),
			OnChanged: func(e Elem) {
				ok = true
			},
		}.Provider()
		scope = scope.Sub(provider)
		scope.Call(func(
			access Access,
		) {
			access(42)
			if access() != 42 {
				t.Fatal()
			}
			if !ok {
				t.Fatal()
			}
		})

		ok = false
		provider = SeriesValue{
			Type:   Elem(0),
			Access: Access(nil),
			OnChanged: func(e Elem) {
				ok = true
			},
		}.Provider()
		scope = scope.Sub(provider)
		scope.Call(func(
			access Access,
		) {
			access()
			if ok {
				t.Fatal()
			}
		})

	})
}
