I've been tooling with a new heap analyzer in Scala for some time.  I've
finally become fed up with the JVM performance.

* None of the GCs can handle the vast tracts of object data I need, and working
  with off-heap memory is slow and painful, despite Java's claims that
  ByteBuffer access are JIT-compiled to native array accesses.

* I keep banging into really slow parts of both Scala and Java like closures
  and abstract methods.  ("Really slow" is relative of course....  the target
  use case for the analyzer is a 10GB heap with 100 million objects.)

* I'd like to add more concurrency and have been impressed with Go's model.

So I'm going to start roughing this out in Go and see where it leads.

