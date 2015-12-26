package stack

type nilItem struct{}

func (i nilItem) Key() string {
	return ""
}

var NilItem = nilItem{}
type StackImpl interface {
	Cap() int
	Len() int
	PopLast()
	Realloc()
}

func stackPop(s StackImpl, n int) {
	if n <= 0 {
		return
	}

	for s.Len() > 0 {
		s.PopLast()
		n--
		if n <= 0 {
			break
		}
	}

	if c := s.Cap(); c > 20 && c > s.Len() * 2 {
		s.Realloc()
	}
}
