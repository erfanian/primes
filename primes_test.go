package main

import (
	"fmt"
	"math/rand"
	"os"
	"syscall"
	"testing"
)

import "os/signal"

import "sync"

import "math/big"

func TestIsEven(t *testing.T) {
	if IsEven(big.NewInt(4)) == false {
		t.Error("Even is odd.")
	}

	if IsEven(big.NewInt(3)) == true {
		t.Error("Odd is even.")
	}

	if IsEven(big.NewInt(0)) == false {
		t.Error("0th bit is 1")
	}

	if IsEven(big.NewInt(-4)) == false {
		t.Error("Even is odd.")
	}

	if IsEven(big.NewInt(-3)) == true {
		t.Error("Odd is even.")
	}
}

func ExampleGenerateCandidates_interrupt() {
	candidates := make(chan *big.Int, 1000)
	var wg sync.WaitGroup
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT)
	wg.Add(1)
	go GenerateCandidates(big.NewInt(5), candidates, &wg, sigs)
	sigs <- syscall.SIGINT
	wg.Wait()
	// Output:
	// Candidate generation halted.
}

func TestFindPrimes(t *testing.T) {
	expectedPrimes := []*big.Int{big.NewInt(5), big.NewInt(7), big.NewInt(11), big.NewInt(13), big.NewInt(17), big.NewInt(19), big.NewInt(23), big.NewInt(29), big.NewInt(31), big.NewInt(37), big.NewInt(41), big.NewInt(43), big.NewInt(47), big.NewInt(53), big.NewInt(59), big.NewInt(61), big.NewInt(67), big.NewInt(71)}
	candidates := make(chan *big.Int, 40)
	candidates <- big.NewInt(9) // Test mod 3
	candidates <- big.NewInt(35)
	for i := range expectedPrimes {
		// We need to verify our results later so do a deep copy here.
		candidates <- new(big.Int).Set(expectedPrimes[i])
		candidates <- big.NewInt(int64(4 * rand.Intn(20))) // Test mod 2
	}
	primes := make(chan *big.Int, 20)
	var wg sync.WaitGroup
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT)

	wg.Add(1)
	go FindPrimes(candidates, primes, &wg, sigs, false)
	close(candidates)
	wg.Wait()
	close(primes)
	for i := 0; i < len(expectedPrimes); i++ {
		lhs := <-primes
		rhs := expectedPrimes[i]
		if lhs.Cmp(rhs) != 0 {
			fmt.Println(lhs.String() + " does not equal " + rhs.String())
			t.Error("Prime mismatch")
		}
	}
}

func TestBubbleSort(t *testing.T) {
	expectedOrder := []*big.Int{big.NewInt(5), big.NewInt(7), big.NewInt(11), big.NewInt(13), big.NewInt(17), big.NewInt(19), big.NewInt(23), big.NewInt(29), big.NewInt(31), big.NewInt(37), big.NewInt(41), big.NewInt(43), big.NewInt(47), big.NewInt(53), big.NewInt(59), big.NewInt(61), big.NewInt(67), big.NewInt(71)}
	actualOrder := []*big.Int{big.NewInt(71), big.NewInt(19), big.NewInt(17), big.NewInt(5), big.NewInt(7), big.NewInt(11), big.NewInt(53), big.NewInt(13), big.NewInt(67), big.NewInt(47), big.NewInt(29), big.NewInt(31), big.NewInt(43), big.NewInt(37), big.NewInt(41), big.NewInt(23), big.NewInt(61), big.NewInt(59)}
	BubbleSort(&actualOrder, false)
	for i := 0; i < len(expectedOrder); i++ {
		lhs := actualOrder[i]
		rhs := expectedOrder[i]
		if lhs.Cmp(rhs) != 0 {
			fmt.Println(lhs.String() + " does not equal " + rhs.String())
			t.Error("Bubble sort failed.")
		}
	}
}
