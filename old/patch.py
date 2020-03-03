#!/usr/bin/python2
import sys
from ptrace.debugger.debugger import PtraceDebugger
from ptrace.debugger.process import PtraceProcess

# infinite loop: _start: nop; nop; xor eax, eax; test eax, eax; je _start
code = '\x90\x90\x31\xc0\x85\xc0\x74\xf8'
stack = 0

dbg = PtraceDebugger()
proc = dbg.addProcess(int(sys.argv[1]), False)
rip = proc.getInstrPointer()
print "[+] Current instruction pointer: %s" % hex(rip)
for map in proc.readMappings():
    if "rwx" in map.permissions:
        stack = map.start
        if stack:
            print "[+] Found RWX mapping: %s" % hex(stack)
            break
if not stack:
    print "[-] No suitable address space found"
    sys.exit(0)
proc.writeBytes(stack, code)
print "[+] Wrote %d bytes to %s" % (len(code), hex(stack))
stack += 2
proc.setInstrPointer(stack)
print "[+] Resuming execution"
proc.cont()
proc.detach()
