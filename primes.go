/* A utility to generate prime numbers.

Example usage:

$ go run primes.go --max_num_primes=100
*/
package main

import "fmt"
import "sync"
import "runtime"
import "os"
import "os/signal"
import "syscall"
import "flag"
import "bufio"

/* This function will generate prime candidates until:
   1. The operating system provides a SIGINT or
   2. The search has reached the max_num_primes specified or
   3. The candidate to test approached ~2^64.
*/
func GenerateCandidates(start_from uint64, candidates chan uint64, wg *sync.WaitGroup, sigs chan os.Signal) {
	defer wg.Done()
	defer close(candidates)
	defer fmt.Println("Candidate generation halted.")
	var MAX_UINT64 = ^uint64(0)
	// Stop short of the true MAX_UINT64 to avoid possible overflow.
	// This method of of preventing overflow is not ideal, but it works for now.
	MAX_UINT64 -= 2
	for {
		select {
		case <-sigs:
			return
		default:
			if start_from+2 > MAX_UINT64 {
				fmt.Println("Reached maximum candidate search size.")
				return
			}

			candidates <- start_from
			start_from += 2
		}
	}
}

func IsEven(num uint64) bool {
	// Check if the lowest order bit is even.
	return num&0x01 == 0
}

/* Check for prime numbers as long as there are new candidates to test.
   This function only implements some of the tests found here:
   https://en.wikipedia.org/wiki/Primality_test
*/
func FindPrimes(c chan uint64, p chan uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	for x := range c {
		if IsEven(x) {
			continue
		}

		if x%3 == 0 {
			continue
		}

		is_prime := true
		var first_denomenator uint64
		// Divide by two (start searching at minimum possible match).
		first_denomenator = x >> 1
		// Always make sure we're testing an odd number.
		first_denomenator = first_denomenator | 0x01

		for i := first_denomenator; i > 1; i -= 2 {
			if x%i == 0 {
				is_prime = false
				break
			}
		}
		// No divisor found (besides 1)
		if is_prime {
			p <- x
		}
	}
	return
}

// Use bubble sort because values will only be slightly out of order in the channel.
func BubbleSort(primes *[]uint64) {
	for i := len(*primes) - 1; i > 1; i-- {
		for j := 0; j < i; j++ {
			if (*primes)[j] > (*primes)[j+1] {
				(*primes)[j], (*primes)[j+1] = (*primes)[j+1], (*primes)[j]
			}
		}
	}
}

func write_primes(p chan uint64, filename *string, wg *sync.WaitGroup, sigs chan os.Signal, max_num_primes uint64,
	buffer_size uint64) {
	defer wg.Done()
	var m []uint64
	var total_primes_found uint64

	f, err := os.OpenFile(*filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	for prime := range p {
		m = append(m, prime)
		total_primes_found += 1
		if total_primes_found >= max_num_primes {
			sigs <- syscall.SIGINT
			fmt.Println("Found max num primes.")
			break
		}
		if uint64(len(m)) >= buffer_size {
			BubbleSort(&m)
			for i := range m {
				fmt.Fprintln(w, m[i])
			}
			m = m[:0]
			w.Flush()
		}
	}
	BubbleSort(&m)
	for i := range m {
		fmt.Fprintln(w, m[i])
	}
	fmt.Println("Wrote", total_primes_found, "primes.")
}

func main() {
	max_num_primes := flag.Uint64("max_num_primes", 1000000, "The maximum number of primes to compute.")
	buffer_size := flag.Uint64("max_buffer", 10000, "The maximum number of values to store in the buffers between operations.")
	filename := flag.String("output_filename", "found_primes.txt", "The file to write prime numbers to.")
	start_from := flag.Uint64("start_from", 5, "The number from which to start searching for primes.")
	max_threads := flag.Int("max_threads", runtime.NumCPU(), "The maximum number of threads to compute primes.")
	flag.Parse()

	candidates := make(chan uint64, *buffer_size)
	primes := make(chan uint64, 2**buffer_size)
	if IsEven(*start_from) {
		panic("--start_from must not be even")
	} else if *start_from <= 5 {
		primes <- 2
		primes <- 3
	}

	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT)

	var wg sync.WaitGroup
	var write_wg sync.WaitGroup

	wg.Add(1)
	write_wg.Add(1)
	go GenerateCandidates(*start_from, candidates, &wg, sigs)
	go write_primes(primes, filename, &write_wg, sigs, *max_num_primes, *buffer_size)
	for i := 0; i < *max_threads; i++ {
		wg.Add(1)
		go FindPrimes(candidates, primes, &wg)
	}
	wg.Wait()
	fmt.Println("Worker threads finished.")
	close(primes)
	write_wg.Wait()
	fmt.Println("Finished writing values to disk.")
}
