/**
 * iterative-permutation.js
 *
 * An iterative form of heap's algorithm. This emulates the algorithm by
 * encoding the program stack in the `stack` variable.  It iteratively unrolls
 * this as new entries are needed.
 *
 * See https://en.wikipedia.org/wiki/Heap's_algorithm
 *
 * License: MIT
 * Author: Brian Card
 */

/**
  * Creates a new Permutation of the given array x. Use the `next` method
  * to get a new permutation and the `hasNext` to check to see if there
  * are any left.
  *
  * @param {Array} x an array of two or more elements
  */
function Permutation (x) {
  this.maxIterations = factorial(x.length)
  this.iterations = 0
  this.x = x
  this.n = x.length
  this.stack = []
  for (var i = this.n; i > 0; i--) {
    this.stack.push({ n: i, index: 0 })
  }
}

/**
* Returns the next element in the permutation.  This will return a copy
* of the array with the elements that are passed in, you are free to modify
* this array.  This may be called after `hasNext` returns false, it will
* repeat the permutation sequence again.
*
* @return an array of elements that are swapped to represent a new ordering
*/
Permutation.prototype.next = function () {
  this.iterations++
  return this.doNext()
}

// helper to perform the next calculation, separated out for clairity
Permutation.prototype.doNext = function () {
  var s = this.stack.pop()
  var skipSwap = false

  while (s.n !== 1) {
    if (!skipSwap) {
      if (s.n % 2 === 0) {
        this.swap(s.index, s.n - 1)
      } else {
        this.swap(0, s.n - 1)
      }
      s.index++
    }

    if (s.index < s.n) {
      this.stack.push(s)
      this.stack.push({ 'n': s.n - 1, index: 0 })
      skipSwap = true
    }

    s = this.stack.pop()
  }

  return this.x.slice(0)
}

// swaps two elements
Permutation.prototype.swap = function (i, j) {
  var tmp = this.x[i]
  this.x[i] = this.x[j]
  this.x[j] = tmp
}

/**
 * Returns `true` if there are more permutations to generate, `false`
 * if all permutations have been exhausted.
 */
Permutation.prototype.hasNext = function () {
  return this.iterations < this.maxIterations
}

/**
 * Returns the total number of permutations avaiable, which is
 * n! for a set of length n.
 */
Permutation.prototype.getTotal = function () {
  return this.maxIterations
}

function factorial (num) {
  var result = num
  while (num > 1) {
    num--
    result = result * num
  }
  return result
}

export default Permutation
