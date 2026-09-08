package sdk

func (e *Engine) ReadFile(vpath string) (string, error) {
	if err := e.checkOpen(); err != nil {
		return "", err
	}
	return e.fs.Read(vpath)
}

func (e *Engine) WriteFile(vpath string, data []byte) (string, error) {
	if err := e.checkOpen(); err != nil {
		return "", err
	}
	return e.fs.Write(vpath, data)
}

func (e *Engine) EditFile(vpath, oldString, newString string, replaceAll bool) (string, error) {
	if err := e.checkOpen(); err != nil {
		return "", err
	}
	return e.fs.Edit(vpath, oldString, newString, replaceAll)
}

func (e *Engine) DeleteFile(vpath string) error {
	if err := e.checkOpen(); err != nil {
		return err
	}
	return e.fs.Delete(vpath)
}

func (e *Engine) ListFiles(vdir string) (ListResult, error) {
	if err := e.checkOpen(); err != nil {
		return ListResult{}, err
	}
	return e.fs.List(vdir)
}

func (e *Engine) StatFile(vpath string) (FileInfo, error) {
	if err := e.checkOpen(); err != nil {
		return FileInfo{}, err
	}
	return e.fs.Stat(vpath)
}
