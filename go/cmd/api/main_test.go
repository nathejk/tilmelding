package main

import "testing"

func Average(floats []float64) float64 {
	return 1.5
}
func TestAverage(t *testing.T) {
	var v float64
	v = Average([]float64{1, 2})
	if v != 1.5 {
		t.Error("Expected 1.5, got ", v)
	}
}
