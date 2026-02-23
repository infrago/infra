package bamgoo

func (Method) RegistryComponent() string {
	return "method"
}

func (Methods) RegistryComponent() string {
	return "method"
}

func (Service) RegistryComponent() string {
	return "service"
}

func (Services) RegistryComponent() string {
	return "service"
}

func (Trigger) RegistryComponent() string {
	return "trigger"
}

func (Library) RegistryComponent() string {
	return "library"
}
