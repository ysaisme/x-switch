#!/usr/bin/env python3
import struct
import zlib
import os

def create_png(size, filepath):
    width = height = size
    
    def make_line(y):
        line = b''
        cx, cy = width / 2, height / 2
        r = width * 0.38
        inner_r = width * 0.22
        for x in range(width):
            dx = x - cx
            dy = y - cy
            dist = (dx*dx + dy*dy) ** 0.5
            
            if dist <= r and dist >= inner_r:
                angle = 0.0
                if dist > 0:
                    import math
                    angle = math.atan2(dy, dx)
                
                if inner_r * 0.8 <= dist <= r * 0.95:
                    if abs(dx) < r * 0.15 and dy < 0:
                        red, green, blue, alpha = 59, 130, 246, 255
                    elif abs(dx) < r * 0.15 and dy > 0:
                        red, green, blue, alpha = 59, 130, 246, 255
                    elif abs(dy) < r * 0.15 and dx > 0:
                        red, green, blue, alpha = 59, 130, 246, 255
                    elif abs(dy) < r * 0.15 and dx < 0:
                        red, green, blue, alpha = 59, 130, 246, 255
                    else:
                        red, green, blue, alpha = 37, 99, 235, 230
                elif dist > r * 0.95:
                    red, green, blue, alpha = 29, 78, 216, 255
                else:
                    red, green, blue, alpha = 30, 64, 175, 200
            elif dist > r:
                red, green, blue, alpha = 0, 0, 0, 0
            else:
                red, green, blue, alpha = 0, 0, 0, 0
            
            line += struct.pack('BBBB', red, green, blue, alpha)
        return line
    
    raw_data = b''
    for y in range(height):
        raw_data += b'\x00'
        raw_data += make_line(y)
    
    def make_chunk(chunk_type, data):
        chunk = chunk_type + data
        crc = struct.pack('>I', zlib.crc32(chunk) & 0xFFFFFFFF)
        return struct.pack('>I', len(data)) + chunk + crc
    
    signature = b'\x89PNG\r\n\x1a\n'
    ihdr_data = struct.pack('>IIBBBBB', width, height, 8, 6, 0, 0, 0)
    ihdr = make_chunk(b'IHDR', ihdr_data)
    
    compressed = zlib.compress(raw_data, 9)
    idat = make_chunk(b'IDAT', compressed)
    iend = make_chunk(b'IEND', b'')
    
    with open(filepath, 'wb') as f:
        f.write(signature + ihdr + idat + iend)

out_dir = os.environ.get('ICONSET_DIR', '/tmp/mswitch.iconset')
os.makedirs(out_dir, exist_ok=True)

sizes = [16, 32, 64, 128, 256, 512]
for s in sizes:
    create_png(s, os.path.join(out_dir, f'icon_{s}x{s}.png'))
    if s <= 256:
        create_png(s*2, os.path.join(out_dir, f'icon_{s}x{s}@2x.png'))

print(f"iconset created in {out_dir}")
