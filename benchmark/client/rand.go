package main

import "math/rand/v2"

var sample [][]byte

func init() {
	sample = append(sample, repeat(200))
	sample = append(sample, repeat(200))
	sample = append(sample, repeat(200))
	sample = append(sample, repeat(200))
	sample = append(sample, repeat(200))
	sample = append(sample, repeat(200))

	//sample = append(sample, repeat(400))
	//sample = append(sample, repeat(500))
	//sample = append(sample, repeat(1000))
	//sample = append(sample, repeat(2000))
	//sample = append(sample, repeat(4000))
}

func getByte() []byte {
	return sample[rand.IntN(len(sample))]
}

func repeat(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = rand.N[byte](4) + 'a'
	}
	return b
}
