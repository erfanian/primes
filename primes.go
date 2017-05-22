/*A utility to generate prime numbers.

Example usage:

$ go run primes.go --maxNumPrimes=100
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

/*GenerateCandidates will generate prime candidates until:
  1. The operating system provides a SIGINT or
  2. The search has reached the maxNumPrimes specified.
*/
func GenerateCandidates(startFrom *big.Int, candidates chan *big.Int, wg *sync.WaitGroup, sigs chan os.Signal) {
	defer wg.Done()
	defer close(candidates)
	defer fmt.Println("Candidate generation halted.")

	for {
		select {
		case <-sigs:
			sigs <- syscall.SIGINT
			return
		default:
			candidates <- new(big.Int).Set(startFrom)
			startFrom.Add(startFrom, big.NewInt(2))
		}
	}
}

//IsEven checks if the lowest order bit is even.
func IsEven(num *big.Int) bool {
	return num.Bit(0) == 0
}

//FirstDenomenator finds ⌊√x⌋, the largest integer such that z² ≤ x, and then
// rounds to the closet odd number.
func FirstDenomenator(num *big.Int) *big.Int {
	firstDenomenator := big.NewInt(0)
	firstDenomenator.Sqrt(num)
	// Always make sure we're testing an odd number.
	return firstDenomenator.Or(firstDenomenator, big.NewInt(1))
}

/*FindPrimes as long as there are new candidates to test.
  This function only implements some of the tests found here:
  https://en.wikipedia.org/wiki/Primality_test
*/
func FindPrimes(c chan *big.Int, p chan *big.Int, wg *sync.WaitGroup, sigs chan os.Signal, useProbablyPrimeFlag bool) {
	defer wg.Done()
	for x := range c {
		if IsEven(x) {
			continue
		}

		if big.NewInt(0).Mod(x, big.NewInt(3)).Cmp(big.NewInt(0)) == 0 {
			continue
		}

		isPrime := true
		stablePrime := new(big.Int).Set(x)

		if useProbablyPrimeFlag {
			if x.Cmp(big.NewInt(math.MaxInt32)) <= 0 {
				// The docs say this will be 100% accurate in this scenario, so we set n = 1.
				if x.ProbablyPrime(1) {
					select {
					case <-sigs:
						sigs <- syscall.SIGINT
						return
					default:
						p <- stablePrime
						// TODO remove continue if we continue in branch below
						continue
					}
				}
				// TODO with low n, it looks like we miss a prime every 1/100000
				// primes. Until we figure out the right n or identify a problem
				// with ProbablyPrime, we fall back to factorization to avoid false
				// negatives.
				// continue
			}
		}

		firstDenomenator := FirstDenomenator(x)

		for i := firstDenomenator; i.Cmp(big.NewInt(1)) == 1; i.Sub(i, big.NewInt(2)) {
			if big.NewInt(0).Mod(x, i).Cmp(big.NewInt(0)) == 0 {
				isPrime = false
				break
			}
		}
		// No divisor found (besides 1)
		if isPrime {
			select {
			case <-sigs:
				sigs <- syscall.SIGINT
				return
			default:
				p <- stablePrime
			}
		}
	}
	return
}

//BubbleSort because values will only be slightly out of order in the channel.
func BubbleSort(primes *[]*big.Int, showPercentage bool) {
	var percentage float64

	for i := len(*primes) - 1; i > 1; i-- {
		if showPercentage {
			percentage = (float64(len(*primes)-i) / float64(len(*primes))) * 100
			fmt.Printf("\r%f%%", percentage)
		}
		for j := 0; j < i; j++ {
			lhs := (*primes)[j]
			rhs := (*primes)[j+1]
			// lhs is > rhs
			if lhs.Cmp(rhs) == 1 {
				(*primes)[j], (*primes)[j+1] = (*primes)[j+1], (*primes)[j]
			}
		}
	}
}

func writePrimes(p chan *big.Int, filename *string, wg *sync.WaitGroup, sigs chan os.Signal, maxNumPrimes *big.Int,
	bufferSize uint64) {
	defer wg.Done()
	var m []*big.Int
	totalPrimesFound := big.NewInt(0)

	f, err := os.OpenFile(*filename+"_presort", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	for prime := range p {
		m = append(m, prime)
		totalPrimesFound.Add(totalPrimesFound, big.NewInt(1))
		if totalPrimesFound.Cmp(maxNumPrimes) >= 0 {
			sigs <- syscall.SIGINT
			fmt.Println("Found max num primes.")
			break
		}

		if uint64(len(m)) >= bufferSize {
			BubbleSort(&m, false)
			// Lines may be slightly out of order becacuse previously written
			// values are not taken into consideration when sorting, so we defer
			// a final sort and write when halting execution.
			for i := range m {
				fmt.Fprintln(w, m[i])
			}
			m = m[:0]
			w.Flush()
		}
	}

	BubbleSort(&m, false)
	for i := range m {
		fmt.Fprintln(w, m[i])
	}
	fmt.Println("Wrote", totalPrimesFound, "primes.")
}

func doFinalSort(filename *string, bufferSize uint64) {
	defer fmt.Println("Finished sorting final output")
	fmt.Println("Sorting final output")
	// We might run out of memory reading this in one-shot, but this can
	// be a future optimization.
	var m []*big.Int
	writeCount := big.NewInt(0)

	f, err := os.OpenFile(*filename+"_presort", os.O_RDONLY, 0660)
	fFinal, finalErr := os.OpenFile(*filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)

	if err != nil {
		panic(err)
	}

	if finalErr != nil {
		panic(finalErr)
	}

	defer f.Close()
	defer fFinal.Close()
	w := bufio.NewWriter(fFinal)
	defer w.Flush()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		scannedInt := big.NewInt(0)
		scannedInt.SetString(scanner.Text(), 10)
		if scannedInt == nil {
			panic("Error reading integer")
		}
		m = append(m, scannedInt)
	}
	if err := scanner.Err(); err != nil {
		panic("Error scanning ints")
	}

	fmt.Println("Doing final bubble sort")
	// TODO replace with merge sort?
	BubbleSort(&m, true)
	fmt.Println("Bubble sort complete")
	// An int64 cast of a uint64 is still a large buffer, so we don't
	// care about truncation in the conversion.
	bigIntBufferSize := big.NewInt(int64(bufferSize))
	for i := range m {
		fmt.Fprintln(w, m[i])
		writeCount.Add(writeCount, big.NewInt(1))

		if writeCount.Cmp(bigIntBufferSize) >= 0 {
			w.Flush()
			writeCount.Set(big.NewInt(0))
		}
	}
	var deleteErr = os.Remove(*filename + "_presort")
	if deleteErr != nil {
		panic(deleteErr)
	}
}

func main() {
	maxNumPrimes := big.NewInt(0)
	startFrom := big.NewInt(0)

	var wg, writeWg sync.WaitGroup
	sigs := make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGINT)

	maxNumPrimesFlag := flag.String("maxNumPrimes", "1000000", "The maximum number of primes to compute.")
	startFromFlag := flag.String("startFrom", "5", "The number from which to start searching for primes.")
	bufferSizeFlag := flag.Uint64("maxBuffer", 10000, "The maximum number of values to store in the buffers between operations.")
	filenameFlag := flag.String("outputFilename", "found_primes", "The file to write prime numbers to.")
	maxThreadsFlag := flag.Uint("maxThreads", uint(runtime.NumCPU()), "The maximum number of threads to compute primes.")
	useProbablyPrimeFlag := flag.Bool("useProbablyPrime", true, "Use the golang ProbablyPrime function. Negatives will automatically fall back to factorization tests.")
	doFinalOutputSortFlag := flag.Bool("doFinalOutputSort", false, "Do a final sort over found primes. Output should already have minimal entropy, ~1/1000000 out of order")

	flag.Parse()

	maxNumPrimes.SetString(*maxNumPrimesFlag, 10)
	startFrom.SetString(*startFromFlag, 10)

	if maxNumPrimes == nil || startFrom == nil {
		panic("Error converting command line values to big Ints.")
	} else if IsEven(startFrom) {
		panic("--startFrom must not be even")
		// Checking for common divisors outweights supporting 2 & 3.
	} else if startFrom.Cmp(big.NewInt(5)) == -1 {
		panic("You must --startFrom at least 5")
	} else if maxNumPrimes.Sign() <= 0 || startFrom.Sign() <= 0 {
		panic("You must supply positive integers.")
	}

	candidates := make(chan *big.Int, *bufferSizeFlag)
	primes := make(chan *big.Int, 2**bufferSizeFlag)

	wg.Add(1)
	writeWg.Add(1)
	go GenerateCandidates(startFrom, candidates, &wg, sigs)
	go writePrimes(primes, filenameFlag, &writeWg, sigs, maxNumPrimes, *bufferSizeFlag)
	for i := uint(0); i < *maxThreadsFlag; i++ {
		wg.Add(1)
		go FindPrimes(candidates, primes, &wg, sigs, *useProbablyPrimeFlag)
	}
	wg.Wait()
	fmt.Println("Worker threads finished.")
	close(primes)
	writeWg.Wait()
	if *doFinalOutputSortFlag {
		doFinalSort(filenameFlag, *bufferSizeFlag)
	}
	fmt.Println("Finished writing values to disk.")
}
