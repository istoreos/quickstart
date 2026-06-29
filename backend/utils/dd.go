package utils

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

func Dd(reader io.Reader, of string, per, size int64) error {
	//fmt.Println("of=", of, "per=", per, "size=", size)
	cmd := exec.Command("dd", "of="+of, "bs="+strconv.FormatInt(per, 10), "count="+strconv.FormatInt((size+per-1)/per, 10))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		io.Copy(stdin, reader)
		stdin.Close()
	}()

	bytes, err := cmd.Output()
	fmt.Println(string(bytes))
	return err
}
