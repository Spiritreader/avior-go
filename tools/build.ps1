param([ValidateSet("windows","linux","all")][string]$Target = "windows")
$env:CGO_ENABLED = "0"
switch ($Target) {
  "windows" { $env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/avior-go-windows-amd64.exe app.go }
  "linux"   { $env:GOOS="linux";   $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/avior-go-linux-amd64 app.go }
  "all"     { & $PSCommandPath -Target windows; & $PSCommandPath -Target linux }
}
