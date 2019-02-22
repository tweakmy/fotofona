package main

import "testing"

func TestSum(t *testing.T) {

    if !FuncA() {
        t.Error("Wrong")
    }

}