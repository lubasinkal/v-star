# v-star ($v^*$)

A high-performance, zero-dependency actuarial engine for Concurrent Financial Simulations built in Go.

## The Origin
The name **v-star** comes from a class joke between my University lecturer and comrades (brothers and sister deployed to study Actuarial Science): 
If an annuity (or more precisely, the payments/premiums associated with the annuity) compound (or earn interest) at rate j while being discounted (valued) at rate i, then the adjusted (effective) discount factor is

$v^* = (1+j)*v$. 

## Why v-star?
Modern financial software is often bloated and slow. **v-star** is designed for:
* **Zero Dependencies:** Uses only the Go Standard Library.
* **Extreme Speed:** Leverages Go's concurrency (goroutines) for mass valuations.
* **Auditability:** Pure, readable math implementations.
* **Possible Job Offers** : Unemployed so why not spend my time testing out how far Go can go in actuarial work

## Installation
```bash
go get [github.com/lubasinkal/v-star](https://github.com/lubasinkal/v-star)
```

### Just dont install yet I'm working on it

