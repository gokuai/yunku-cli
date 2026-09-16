package edition

// defaultHooks returns the open-source edition defaults.
// All function hooks are nil, which the internal code interprets as
// "use standard open-source behaviour".
func defaultHooks() *Hooks {
	return &Hooks{
		Name: "open",
	}
}
