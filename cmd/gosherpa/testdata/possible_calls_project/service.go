package possible

func Entry(callback func()) {
	go Target()
	callback()
}

func Target() {}
