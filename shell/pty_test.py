import os
import pty
from pathlib import Path
import select
import signal
import tempfile
import time

root = str(Path(__file__).resolve().parent.parent)
with tempfile.TemporaryDirectory() as test_dir:
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.update(TERM='xterm-256color', DEJA_BIN=os.environ.get('DEJA_TEST_BIN', root+'/bin/deja'),
                          DEJA_STORE=test_dir+'/history.jsonl', TMPDIR=test_dir)
        os.execvp('zsh', ['zsh', '-dfi'])
    transcript = bytearray()
    def until(marker, seconds=8):
        if marker in transcript:
            return
        deadline = time.monotonic()+seconds
        while time.monotonic() < deadline:
            if select.select([fd], [], [], .1)[0]:
                transcript.extend(os.read(fd, 65536))
                if marker in transcript:
                    return
        raise AssertionError('missing '+repr(marker)+'; terminal='+repr(bytes(transcript[-5000:])))
    try:
        setup = "unset HISTFILE; \"$DEJA_BIN\" record --timestamp 100 git status; source '"+root+"/shell/deja.zsh'; function show_buffer() { zle -M -- \"INSERTED:$BUFFER\"; }; zle -N show_buffer; bindkey '^X^B' show_buffer; PROMPT='TEST_READY> '\n"
        os.write(fd, setup.encode())
        until(b'TEST_READY> \x1b[K')
        transcript.clear()
        os.write(fd, b'git')
        until(b'distinct variants')
        until(b'[status]')
        transcript.clear()
        os.write(fd, b'\t\x18\x02')
        until(b'INSERTED:git status')
        os.write(fd, b'\x03exit\n')
        print('real ZLE: async display without extra keystroke and Tab insertion passed')
    finally:
        os.close(fd)
        try:
            os.kill(pid, signal.SIGHUP)
        except ProcessLookupError:
            pass
        os.waitpid(pid, 0)
