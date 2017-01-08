package main

import "testing"
import "os"
import "os/signal"
import "syscall"
import "sync"
import "math/rand"

func TestIsEven(t *testing.T) {
	if IsEven(4) == false {
		t.Error("Even is odd.")
	}

	if IsEven(3) == true {
		t.Error("Odd is even.")
	}
}

func ExampleGenerateCandidatesInterrupt() {
	candidates := make(chan uint64, 1000)
	var wg sync.WaitGroup
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT)
	wg.Add(1)
	go GenerateCandidates(5, candidates, &wg, sigs)
	sigs <- syscall.SIGINT
	wg.Wait()
	// Output:
	// Candidate generation halted.
}

func ExampleGenerateCandidatesUpperBound() {
	candidates := make(chan uint64, 1000)
	var wg sync.WaitGroup
	sigs := make(chan os.Signal, 3)

	var MAX_UINT64 = ^uint64(0)
	MAX_UINT64 -= 10
	wg.Add(1)
	go GenerateCandidates(MAX_UINT64, candidates, &wg, sigs)
	wg.Wait()
	// Output:
	// Reached maximum candidate search size.
	// Candidate generation halted.
}

func TestFindPrimes(t *testing.T) {
	expected_primes := []uint64{5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	candidates := make(chan uint64, 38)
	candidates <- 9 // Test mod 3
	candidates <- 35
	for i := range expected_primes {
		candidates <- expected_primes[i]
		candidates <- uint64(4 * rand.Intn(20)) // Test mod 2
	}
	primes := make(chan uint64, 20)
	var wg sync.WaitGroup

	wg.Add(1)
	go FindPrimes(candidates, primes, &wg)
	close(candidates)
	wg.Wait()
	close(primes)
	for i := 0; i < len(expected_primes); i++ {
		p := <-primes
		if p != expected_primes[i] {
			t.Error("Prime mismatch")
		}
	}
}

func TestBubbleSort(t *testing.T) {
	expected_order := []uint64{5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71}
	actual_order := []uint64{71, 19, 17, 5, 7, 11, 53, 13, 67, 47, 29, 31, 43, 37, 41, 23, 61, 59}
	BubbleSort(&actual_order)
	for i := 0; i < len(expected_order); i++ {
		if actual_order[i] != expected_order[i] {
			t.Error("Bubble sort failed.")
		}
	}
}
