package trypkg

import "testing"

func TestScriptRename(t *testing.T) {
	cmds := scriptRename("/tmp/tries", "old", "new")
	if len(cmds) != 4 {
		t.Fatalf("unexpected len: %d", len(cmds))
	}
	if cmds[1] != "mv 'old' 'new'" {
		t.Fatalf("unexpected mv command: %s", cmds[1])
	}
}

func TestScriptDelete(t *testing.T) {
	cmds := scriptDelete([]string{"a", "b"}, "/tmp/tries")
	if len(cmds) != 4 {
		t.Fatalf("unexpected len: %d", len(cmds))
	}
	if cmds[1] != "test -d 'a' && rm -rf 'a'" {
		t.Fatalf("unexpected delete command: %s", cmds[1])
	}
}
