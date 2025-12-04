package main
import (
  "fmt"
  "brabble/internal/config"
)
func main(){
  cfg, err := config.Load("")
  if err!=nil { panic(err)}
  fmt.Printf("hooks=%d hook.command=%q\n", len(cfg.Hooks), cfg.Hook.Command)
  for i,h:= range cfg.Hooks {
    fmt.Printf("hook %d wake=%v aliases=%v cmd=%s args=%v\n", i, h.Wake, h.Aliases, h.Command, h.Args)
  }
}
