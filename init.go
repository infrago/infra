package infra

func init() {
	Mount(core)
	Mount(basic)
	Mount(codec)
	Mount(builtin)
	Mount(library)
	Mount(trigger)

	hook.AttachBus(&defaultBusHook{})
	hook.AttachConfig(&defaultConfigHook{})
	hook.AttachTrace(&defaultTraceHook{})
}
