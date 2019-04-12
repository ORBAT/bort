# bort

Instead of writing a bad sorting algorithm myself, I figured I'd evolve one. 

## What

`bort` is a genetic programming system that generates individuals that sort integer lists, or try to
at any rate. At its core is a stack-based virtual machine, and genomes (or "critters" in the source
code) that are programs for that VM. Critters evolve by crossing over parts of their genetic code
with others using tournament selection and preferring to mate with dissimilar individuals.

### CPU

`bort`'s CPU has 3 stacks:
- exec: this is where executable code lives. Code is run by popping instructions off `exec`
- int: integer stack
- bool: boolean stack

There are also two registers, Int Stack Pointer 1 and 2, or ISP1 and ISP2.

The instruction set is:
- `incr_isp1`/`2`: increment ISP1/2
- `decr_isp1`/`2`: decrement ISP1/2
- `lt_isps`: push `intStack[isp1 % len(instStack)] < intStack[isp2 % len(instStack)]`
- `swap_isps`: swap `intStack[isp1 % len(instStack)]` and `intStack[isp2 % len(instStack)]` 
- `nop`: do nothing
- `sorted`: if the int stack is sorted, push true onto the bool stack, false otherwise
- `halt`: stop execution
- `y`: Y-combinator-but-not-really. y basically copies the exec stack, and prepends the copy (along
with itself) back to exec, creating a loop (in Forth stack notation, this is `( a b c d y -- a b c d
y a b c d )`)
- `if`: executes either of the previous two commands depending on if the top of the bool stack is
true or not


## Installation & usage

- install Go >=1.12
- `go install github.com/ORBAT/bort`

To run, pass in a comma-delimited list of numbers:
- `bort 1,6,2,5,7,3,1` (or `bort -verbose 5,6,1,4` for more spam). You'll eventually see how well
the best evolved sorter did with your array.

Note that you may have to run `bort` several times, and it might not be able to sort your array.