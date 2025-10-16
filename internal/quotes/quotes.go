// Package quotes provides a pool of wisdom quotes.
// Random() returns a random quote using crypto/rand.
package quotes

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// pool is a hardcoded list of wisdom quotes.
var pool = []string{
	"The only true wisdom is in knowing you know nothing. – Socrates",
	"The journey of a thousand miles begins with one step. – Lao Tzu",
	"In the middle of difficulty lies opportunity. – Albert Einstein",
	"Life is what happens when you're busy making other plans. – John Lennon",
	"The only impossible journey is the one you never begin. – Tony Robbins",
	"Do not dwell in the past, do not dream of the future, concentrate the mind on the present moment. – Buddha",
	"The greatest glory in living lies not in never falling, but in rising every time we fall. – Nelson Mandela",
	"You must be the change you wish to see in the world. – Mahatma Gandhi",
	"Success is not final, failure is not fatal: it is the courage to continue that counts. – Winston Churchill",
	"It does not matter how slowly you go as long as you do not stop. – Confucius",
}

// Random returns a random quote from the pool using crypto/rand.
func Random() (string, error) {
	n := len(pool)
	if n == 0 {
		return "", fmt.Errorf("quote pool is empty")
	}

	// Generate a random index using crypto/rand
	maxBig := big.NewInt(int64(n))
	idx, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		return "", fmt.Errorf("failed to generate random index: %w", err)
	}

	return pool[idx.Int64()], nil
}
