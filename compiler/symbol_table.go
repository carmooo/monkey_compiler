package compiler

type SymbolScope string

const (
	GlobalScope  SymbolScope = "GLOBAL"
	LocalScope   SymbolScope = "LOCAL"
	BuiltInScope SymbolScope = "BUILTIN"
	FreeScope    SymbolScope = "FREE"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer       *SymbolTable
	FreeSymbols []Symbol

	store          map[string]Symbol
	numDefinitions int
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	return &SymbolTable{store: s}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer

	return s
}

func (st *SymbolTable) Define(name string) Symbol {
	sym := Symbol{
		Name:  name,
		Index: st.numDefinitions,
	}

	if st.Outer == nil {
		sym.Scope = GlobalScope
	} else {
		sym.Scope = LocalScope
	}

	st.store[name] = sym
	st.numDefinitions++
	return sym
}

func (st *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	sym := Symbol{
		Name:  name,
		Scope: BuiltInScope,
		Index: index,
	}
	st.store[name] = sym
	return sym
}

func (st *SymbolTable) Resolve(name string) (Symbol, bool) {
	sym, ok := st.store[name]

	if !ok && st.Outer != nil {
		sym, ok := st.Outer.Resolve(name)
		if !ok {
			return sym, ok
		}

		switch sym.Scope {
		case GlobalScope, BuiltInScope:
			return sym, ok
		default:
			free := st.defineFree(sym)
			return free, true
		}

	}
	return sym, ok
}

func (st *SymbolTable) defineFree(original Symbol) Symbol {
	st.FreeSymbols = append(st.FreeSymbols, original)

	symbol := Symbol{Name: original.Name, Index: len(st.FreeSymbols) - 1, Scope: FreeScope}
	st.store[original.Name] = symbol

	return symbol
}
