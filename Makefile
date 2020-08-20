.PHONY: cross
cross:
	@mkdir build
	@GOOS=drawin GOARCH=amd64 go build -ldflags "-s -w" -o build/gosearch_drawin_amd64
	@GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o build/gosearch_windows_amd64.exe
	@GOOS=windows GOARCH=i386 go build -ldflags "-s -w" -o build/gosearch_windows_i386.exe
	@GOOS=linux GOARCH=linux go build -ldflags "-s -w" -o build/gosearch_drawin_amd64

