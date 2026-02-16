# Appendix B: Token Language Grammar

## S-Expression Syntax Reference

<!-- Complete grammar -->

```
net     := (net <name> <stmt>*)
stmt    := <cell> | <func> | <arrow> | <guard>
cell    := (cell <name> <initial>?)
func    := (fn <name>)
arrow   := (arc <source> <target> <weight>?)
guard   := (guard <transition> <expression>)
```

## Guard Expression Language

<!-- Operators and semantics -->
