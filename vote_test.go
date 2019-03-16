package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"

	"github.com/Sam-Izdat/govote"
	"github.com/stretchr/testify/require"
)

func democrats() map[string]int {
	return map[string]int{
		"Biden":        29,
		"Sanders":      20,
		"Harris":       5,
		"O'Rourke":     4,
		"Warren":       4,
		"Booker":       3,
		"Delaney":      3,
		"Klobuchar":    3,
		"Castro":       1,
		"Gabbard":      1,
		"Gillibrand":   1,
		"Hickenlooper": 1,
		"Inslee":       1,
		"Yang":         1,
		"Buttigieg":    0,
		"Williamson":   0,
	}
}

func candidateList(weights map[string]int) []string {
	out := make([]string, 0, len(weights))
	for name := range weights {
		out = append(out, name)
	}
	return out
}

func weighted(weights map[string]int) []*string {
	i := 0
	length := 100 + len(weights)
	result := make([]*string, 0, length)
	for name, v := range weights {
	REDO:
		n := name // BUG: loop capture?
		fmt.Printf("adding %s at %d\n", name, i)
		result = append(result, &n)
		i++
		v--
		if v > 0 {
			goto REDO
		}
	}
	return result
}

func randomBallot(random *rand.Rand, choices []*string, min, max int) []string {
	if min > max {
		panic("min > max")
	}
	if min < 1 {
		panic("min < 1")
	}
	if max > len(choices) {
		panic("max > len(choices)")
	}

	count := random.Intn(max-min+1) + min
	votes := make(map[string]bool, count)
	output := make([]string, count, count)
	for i := 0; i < count; i++ {
	RETRY: // suboptimal but better than building a giant decision tree in memory since we don't expect more than a handful of votes per
		index := random.Intn(len(choices))
		vote := *choices[index]
		if exists := votes[vote]; exists {
			//fmt.Printf("exists %d %s\n", index, vote)
			goto RETRY
		}
		//fmt.Printf("ranked %d %s\n", index, vote)
		votes[vote] = true
		output[i] = vote
	}
	return output
}

func TestVeryMany(t *testing.T) {

	random := rand.New(rand.NewSource(2019))

	var a string = "Hello"
	var b string = "Hello"

	const (
		population = 328285992
		buffer     = 1 << 16
	)

	require.Equal(t, a, b, "The two words should be the same.")
	dems := democrats()
	candidates := candidateList(dems)
	deck := weighted(dems)
	poll, _ := govote.Schulze.New(candidates)
	// poll.AddBallot([]string{"Kang"})
	// poll.AddBallot([]string{"Kang"})
	// poll.AddBallot([]string{"Kodos"})

	length := runtime.GOMAXPROCS(0)
	c := make(chan []string, length)
	quitters := make([]chan bool, length)
	for i := 0; i < length; i++ {
		quitters[i] = make(chan bool)
		s := rand.NewSource(random.Int63())
		random := rand.New(s)
		go func(quit chan bool) {
		RETRY:
			ballot := randomBallot(random, deck, 1, len(candidates))
			select {
			case <-quit:
				//fmt.Println("voter quitting")
			case c <- ballot:
				goto RETRY
			}
		}(quitters[i])
	}

	for i := 0; i < population/10; i++ {
		ballot := <-c
		if i%1000000 == 0 {
			fmt.Println(i, ballot)
		}
		poll.AddBallot(ballot)
	}
	for i := 0; i < length; i++ {
		quitters[i] <- true
	}
	close(c)
	fmt.Println("calculating")
	result, scores, err := poll.Evaluate()
	if err != err {
		panic(err)
	}
	s := "no result"
	if len(result) > 1 {
		s = "draw"
	} else if len(result) == 1 {
		s = "winner"
	}
	fmt.Println(s, " ", result)
	for i, score := range scores {
		fmt.Printf("% 3.d % 29s\t% 4d\n", i+1, score.Name, score.Score)
	}
}
