#!./alatty/launcher/alatty +launch
# License: GPLv3 Copyright: 2021, Kovid Goyal <kovid at kovidgoyal.net>


import os
import sys
from typing import List

from alatty.conf.generate import write_output

if __name__ == '__main__' and not __package__:
    import __main__
    __main__.__package__ = 'gen'
    sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def main(args: List[str]=sys.argv) -> None:
    from alatty.options.definition import definition
    write_output('alatty', definition)
    nullable_colors = []
    all_colors = []
    for opt in definition.iter_all_options():
        if callable(opt.parser_func):
            if opt.parser_func.__name__ in ('to_color_or_none', 'cursor_text_color'):
                nullable_colors.append(opt.name)
                all_colors.append(opt.name)
            elif opt.parser_func.__name__ in ('to_color', 'titlebar_color', 'macos_titlebar_color'):
                all_colors.append(opt.name)


if __name__ == '__main__':
    import runpy
    m = runpy.run_path(os.path.dirname(os.path.abspath(__file__)))
    m['main']([sys.executable, 'config'])
