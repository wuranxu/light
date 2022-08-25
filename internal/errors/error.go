package errors

import "fmt"

type Err string

func (e Err) New(s interface{}) Err {
	return Err(fmt.Sprintf("%v: %v", e, s))
}

func (e Err) Error() string {
	return string(e)
}
