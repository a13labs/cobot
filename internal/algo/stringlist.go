package algo

// This provide a simple string list
type StringList []string

func (l *StringList) Add(s string) {
	*l = append(*l, s)
}

func (l *StringList) Remove(s string) {
	for i, v := range *l {
		if v == s {
			*l = append((*l)[:i], (*l)[i+1:]...)
			return
		}
	}
}

func (l *StringList) Contains(s string) bool {
	for _, v := range *l {
		if v == s {
			return true
		}
	}
	return false
}

func (l *StringList) Len() int {
	return len(*l)
}

func (l *StringList) Get(i int) string {
	return (*l)[i]
}
