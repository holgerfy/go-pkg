package errno

import "fmt"

type Errno struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg"`
	IsNetErr uint8  `json:"is_net"`
}

func (err *Errno) Error() string {
	return fmt.Sprintf(`{"msg":"%s","code":%d,"is_net":"%d"}`, err.Msg, err.Code, err.IsNetErr)
}

func Add(msg string, code int) error {
	return &Errno{
		Code:     code,
		Msg:      msg,
		IsNetErr: 0,
	}
}

func AddSysErr(msg string, code int) error {
	return &Errno{
		Code:     code,
		Msg:      msg,
		IsNetErr: 1,
	}
}
