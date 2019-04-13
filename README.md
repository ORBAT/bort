# bort

Instead of writing a bad sorting algorithm myself, I figured I'd evolve one. 

## What

`bort` is a genetic programming system that generates individuals that sort integer lists (or try to
at any rate.) At its core is a stack-based virtual machine, and genomes (or "critters" in the source
code) that are programs for that VM.

On a general level, the way `bort` works is that during each "generation", each individual program
tries to sort a randomly generated array, and is then assigned a "fitness" value based on how close
to being sorted the array is. In the next phase, most of the individuals "mate": partners are selected (with
the most fit individuals being [more likely to be
selected](https://en.wikipedia.org/wiki/Tournament_selection)) and then two offspring are [generated
using the parents' code](https://en.wikipedia.org/wiki/Crossover_(genetic_algorithm). Each offspring
also potentially undergoes mutations.

## Why

I've always been interested in evolutionary methods, and `bort` is the second genetic programming project I've done in Go. Like with the previous one, eventually I hope to [produce sound with `bort`](https://soundcloud.com/thomas-oakley-2/). 

## Installation & usage

- install Go >=1.12
- `go install github.com/ORBAT/bort` (`bort -h` gives instructions)

To run, pass in a comma-delimited list of numbers: `bort 1,6,2,5,7,3,1` (or `bort -verbose 5,6,1,4`
for more spam). You'll eventually see how well the best evolved sorter did with your array.

Note that you may have to run `bort` several times, and it might not be able to sort your array.

## CPU

`bort`'s CPU has 3 stacks:
- exec: this is where executable code lives. Code is run by popping instructions off `exec`
- int: integer stack. This is where the input is written, and where the output is read from
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