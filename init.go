package infra

func init() {
	Mount(core)
	Mount(basic)
	Mount(codec)
	Mount(library)
	Mount(trigger)

	hook.AttachBus(&defaultBusHook{})
	hook.AttachConfig(&defaultConfigHook{})
	hook.AttachLog(newDefaultLogHook())
	hook.AttachTrace(&defaultTraceHook{})
	hook.AttachToken(newDefaultTokenHook())
}
