// Copied from https://cs.opensource.google/go/x/exp/+/24438e51023af3bfc1db8aed43c1342817e8cfcd:rand/example_test.go

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rand_test

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"go.mongodb.org/mongo-driver/v2/internal/rand"
)

// These tests serve as an example but also make sure we don't change
// the output of the random number generator when given a fixed seed.

func Example() {
	rand.Seed(42) // Try changing this number!
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}
	fmt.Println("Magic 8-Ball says:", answers[rand.Intn(len(answers))])
	// Output: Magic 8-Ball says: Most likely
}

// This example shows the use of each of the methods on a *Rand.
// The use of the global functions is the same, without the receiver.
func Example_rand() {
	// Create and seed the generator.
	// Typically a non-fixed seed should be used, such as time.Now().UnixNano().
	// Using a fixed seed will produce the same output on every run.
	r := rand.New(rand.NewSource(1234))

	// The tabwriter here helps us generate aligned output.
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	defer w.Flush()
	show := func(name string, v1, v2, v3 interface{}) {
		fmt.Fprintf(w, "%s\t%v\t%v\t%v\n", name, v1, v2, v3)
	}

	// Float32 and Float64 values are in [0, 1).
	show("Float32", r.Float32(), r.Float32(), r.Float32())
	show("Float64", r.Float64(), r.Float64(), r.Float64())

	// ExpFloat64 values have an average of 1 but decay exponentially.
	show("ExpFloat64", r.ExpFloat64(), r.ExpFloat64(), r.ExpFloat64())

	// NormFloat64 values have an average of 0 and a standard deviation of 1.
	show("NormFloat64", r.NormFloat64(), r.NormFloat64(), r.NormFloat64())

	// Int31, Int63, and Uint32 generate values of the given width.
	// The Int method (not shown) is like either Int31 or Int63
	// depending on the size of 'int'.
	show("Int31", r.Int31(), r.Int31(), r.Int31())
	show("Int63", r.Int63(), r.Int63(), r.Int63())
	show("Uint32", r.Uint32(), r.Uint32(), r.Uint32())
	show("Uint64", r.Uint64(), r.Uint64(), r.Uint64())

	// Intn, Int31n, Int63n and Uint64n limit their output to be < n.
	// They do so more carefully than using r.Int()%n.
	show("Intn(10)", r.Intn(10), r.Intn(10), r.Intn(10))
	show("Int31n(10)", r.Int31n(10), r.Int31n(10), r.Int31n(10))
	show("Int63n(10)", r.Int63n(10), r.Int63n(10), r.Int63n(10))
	show("Uint64n(10)", r.Uint64n(10), r.Uint64n(10), r.Uint64n(10))

	// Perm generates a random permutation of the numbers [0, n).
	show("Perm", r.Perm(5), r.Perm(5), r.Perm(5))
	// Output:
	// Float32     0.030719291          0.47512934           0.031019364
	// Float64     0.6906635660087743   0.9898818576905045   0.2683634639782333
	// ExpFloat64  1.24979080914592     0.3451975160045876   0.5456817760595064
	// NormFloat64 0.879221333732727    -0.01508980368383761 -1.962250558270421
	// Int31       2043816560           1870670250           1334960143
	// Int63       7860766611810691572  1466711535823962239  3836585920276818709
	// Uint32      2051241581           751073909            1353986074
	// Uint64      10802154207635843641 14398820303406316826 11052107950969057042
	// Intn(10)    3                    0                    1
	// Int31n(10)  3                    8                    1
	// Int63n(10)  4                    6                    0
	// Uint64n(10) 2                    9                    4
	// Perm        [1 3 4 0 2]          [2 4 0 3 1]          [3 2 0 4 1]
}

func ExampleShuffle() {
	words := strings.Fields("ink runs from the corners of my mouth")
	rand.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})
	fmt.Println(words)

	// Output:
	// [ink corners of from mouth runs the my]
}

func ExampleShuffle_slicesInUnison() {
	numbers := []byte("12345")
	letters := []byte("ABCDE")
	// Shuffle numbers, swapping corresponding entries in letters at the same time.
	rand.Shuffle(len(numbers), func(i, j int) {
		numbers[i], numbers[j] = numbers[j], numbers[i]
		letters[i], letters[j] = letters[j], letters[i]
	})
	for i := range numbers {
		fmt.Printf("%c: %c\n", letters[i], numbers[i])
	}

	// Output:
	// D: 4
	// A: 1
	// E: 5
	// B: 2
	// C: 3
}

func ExampleLockedSource() {
	r := rand.New(new(rand.LockedSource))
	r.Seed(42) // Try changing this number!
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}
	fmt.Println("Magic 8-Ball says:", answers[r.Intn(len(answers))])
	// Output: Magic 8-Ball says: Most likely
}
