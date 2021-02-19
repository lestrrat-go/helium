// +build ignore


package helium



type stack []interface{}

func (s *stack) Push(v interface{}) {
	*data = append(*data, v)
}

func (s *stack) Pop() interface{} {
	l := len(*data)
	if l == 0 {
		return nil
	}
	v := (*data)[l-1]
	*data = (*data)[:l-1]
	return v
}

func (s *stack) Top() interface{} {
	l := len(*data) 
	if l == 0 {
		return nil
	}
	return (*data)[l-1]
}

type CursorStack struct {
	stack *stack
}

func NewCursorStack() *CursorStack {
	return &CursorStack{}
}

func (s *CursorStack) Pop() interf
