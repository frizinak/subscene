package fuzzy

import (
	"sort"
	"strings"
)

const fuzzyLength = 2

type Swapper interface{ Swap(i, j int) }

type sortCol struct {
	b sort.IntSlice
	x []Swapper
}

func (s *sortCol) Swap(i, j int) {
	s.b.Swap(i, j)
	for n := range s.x {
		s.x[n].Swap(i, j)
	}
}
func (s *sortCol) Less(i, j int) bool { return s.b[i] >= s.b[j] }
func (s *sortCol) Len() int           { return len(s.b) }

func Search(q string, list []string, toSort ...Swapper) (scores []int) {
	scores = make([]int, len(list))
	index := make(map[string][]int, len(list))
	for i, v := range list {
		fuzzy := parts(v)
		for _, p := range fuzzy {
			index[p] = append(index[p], i)
		}
	}

	words := parts(q)
	for _, q := range words {
		done := make(map[int]struct{})
		if b, ok := index[q]; ok {
			for i := 0; i < len(b); i++ {
				if _, ok := done[b[i]]; ok {
					continue
				}
				ix := b[i]
				scores[ix]++
				done[b[i]] = struct{}{}
			}
		}
	}

	s := &sortCol{b: sort.IntSlice(scores), x: toSort}
	sort.Sort(s)
	return scores
}

func parts(q string) []string {
	qs := make([]string, 0, len(q))
	p := strings.Fields(
		strings.Trim(strings.TrimSpace(strings.ToLower(q)), "!@#$%^&*=./,"),
	)
	for i := range p {
		if len(p[i]) < 2 {
			continue
		}
		if len(p[i]) <= fuzzyLength {
			qs = append(qs, p[i])
			continue
		}
		for j := 0; j < len(p[i])-fuzzyLength+1; j++ {
			qs = append(qs, strings.TrimSpace(p[i][j:j+fuzzyLength]))
		}
	}

	return qs
}
