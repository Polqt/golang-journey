
# Go Concurrency: Goroutines & Channels

Concurrency is one of Go’s superpowers. In this section, we explore how goroutines and channels let us write fast, parallel algorithms that are easy to reason about.

---

## Goroutines & Channels

- **Goroutines** are lightweight threads managed by Go. You can launch thousands of them!
- **Channels** provide safe, easy communication between goroutines.

Together, they make it simple to parallelize work and coordinate results.

---

## Example: Concurrent Coin Change Algorithm

Let’s outline a classic algorithm (making change) and see how concurrency can help:

**Algorithm Steps:**
1. Sort coin denominations in descending order (prioritize large coins).
2. Initialize an empty `change` slice to store selected coins.
3. For each coin in the sorted list:
	- If the coin fits (amount >= coin):
		- Subtract its value from the amount.
		- Add the coin to `change`.
	- Repeat until the amount is zero or no more coins fit.
4. If the amount is zero, return `change` as the solution.
5. If not, return an empty slice (no solution).

---

In Go, you can parallelize parts of this process (e.g., trying different coin combinations) using goroutines and channels for even more power!