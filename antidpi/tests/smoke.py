#!/usr/bin/env python3
"""Host smoke check: strategy parsing, SOCKS readiness, and HTTP forwarding."""
import http.server
import json
from pathlib import Path
import socket
import struct
import subprocess
import threading
import time

root = Path(__file__).resolve().parents[2]
engine = root / 'antidpi/vendor/ciadpi'
presets = json.loads((root / 'gost/gost_helper/strategies.json').read_text())


def exact(conn, count):
    data = b''
    while len(data) < count:
        chunk = conn.recv(count - len(data))
        if not chunk:
            raise RuntimeError('unexpected EOF')
        data += chunk
    return data


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'antidpi-smoke-ok')

    def log_message(self, *_):
        pass


server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
threading.Thread(target=server.serve_forever, daemon=True).start()
try:
    for name, groups in presets.items():
        with socket.socket() as reserve:
            reserve.bind(('127.0.0.1', 0))
            port = reserve.getsockname()[1]
        args = [str(engine), '--ip', '127.0.0.1', '--port', str(port), '--no-udp', '--auto-mode', '3']
        for group in groups:
            group = list(group)
            if group and group[0].startswith('--auto='):
                args.append(group.pop(0))
            args += ['--proto', 'http,tls'] + group
        proc = subprocess.Popen(args, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        try:
            for attempt in range(100):
                if proc.poll() is not None:
                    raise RuntimeError(proc.stderr.read().decode())
                try:
                    conn = socket.create_connection(('127.0.0.1', port), timeout=.2)
                    break
                except OSError:
                    time.sleep(.02)
            else:
                raise RuntimeError('startup timeout')
            with conn:
                conn.settimeout(3)
                conn.sendall(b'\x05\x01\x00')
                assert exact(conn, 2) == b'\x05\x00'
                if name == 'off':
                    conn.sendall(b'\x05\x01\x00\x01\x7f\x00\x00\x01' + struct.pack('!H', server.server_port))
                    reply = exact(conn, 4)
                    assert reply[:3] == b'\x05\x00\x00', reply
                    assert reply[3] == 1, reply
                    exact(conn, 6)
                    conn.sendall(b'GET / HTTP/1.0\r\nHost: localhost\r\n\r\n')
                    response = b''
                    while chunk := conn.recv(4096):
                        response += chunk
                    assert b'antidpi-smoke-ok' in response
            print(name + ': OK')
        finally:
            proc.terminate()
            proc.communicate(timeout=5)
finally:
    server.shutdown()
