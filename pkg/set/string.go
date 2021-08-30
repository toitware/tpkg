// Copyright (C) 2021 Toitware ApS. All rights reserved.

package set

type String map[string]struct{}

func NewString(strs ...string) String {
	res := String{}
	for _, s := range strs {
		res[s] = struct{}{}
	}
	return res
}

func (s *String) Add(strs ...string) {
	if *s == nil {
		*s = String{}
	}

	for _, str := range strs {
		(*s)[str] = struct{}{}
	}
}

func (s String) Remove(strs ...string) {
	if s == nil {
		return
	}

	for _, str := range strs {
		delete(s, str)
	}
}

func (s String) Contains(str string) bool {
	if s == nil {
		return false
	}

	_, exists := s[str]
	return exists
}

func (s String) Values() []string {
	var res []string
	if s == nil {
		return res
	}

	for k := range s {
		res = append(res, k)
	}
	return res
}
