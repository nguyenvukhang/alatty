#!/usr/bin/env python
# License: GPL v3 Copyright: 2016, Kovid Goyal <kovid at kovidgoyal.net>

import json
import os
from contextlib import contextmanager, suppress
from functools import partial
from typing import Any, Dict, Generator, Iterable, List, Optional

from .conf.utils import BadLine, parse_config_base
from .conf.utils import load_config as _load_config
from .constants import cache_dir
from .options.types import Options, defaults
from .options.utils import KeyboardMode, KeyboardModeMap, KeyDefinition, MouseMap, MouseMapping, build_action_aliases
from .typing import TypedDict
from .utils import log_error


def build_ansi_color_table(opts: Optional[Options] = None) -> int:
    if opts is None:
        opts = defaults
    addr, length = opts.color_table.buffer_info()
    if length != 256 or opts.color_table.typecode != 'L':
        raise TypeError(f'The color table has incorrect size length: {length} typecode: {opts.color_table.typecode}')
    return addr


def atomic_save(data: bytes, path: str) -> None:
    import shutil
    import tempfile
    path = os.path.realpath(path)
    fd, p = tempfile.mkstemp(dir=os.path.dirname(path), suffix='.tmp')
    try:
        with os.fdopen(fd, 'wb') as f:
            f.write(data)
        with suppress(FileNotFoundError):
            shutil.copystat(path, p)
        os.utime(p)
        os.replace(p, path)
    finally:
        try:
            os.remove(p)
        except FileNotFoundError:
            pass
        except Exception as err:
            log_error(f'Failed to delete temp file {p} for atomic save with error: {err}')


@contextmanager
def cached_values_for(name: str) -> Generator[Dict[str, Any], None, None]:
    cached_path = os.path.join(cache_dir(), f'{name}.json')
    cached_values: Dict[str, Any] = {}
    try:
        with open(cached_path, 'rb') as f:
            cached_values.update(json.loads(f.read().decode('utf-8')))
    except FileNotFoundError:
        pass
    except Exception as err:
        log_error(f'Failed to load cached in {name} values with error: {err}')

    yield cached_values

    try:
        data = json.dumps(cached_values).encode('utf-8')
        atomic_save(data, cached_path)
    except Exception as err:
        log_error(f'Failed to save cached values with error: {err}')


def finalize_keys(opts: Options, accumulate_bad_lines: Optional[List[BadLine]] = None) -> None:
    defns: List[KeyDefinition] = []
    for d in opts.map:
        if d is None:  # clear_all_shortcuts
            defns = []  # type: ignore
        else:
            try:
                defns.append(d.resolve_and_copy(opts.alatty_mod))
            except Exception as err:
                if accumulate_bad_lines is None:
                    log_error(f'Ignoring map with invalid action: {d.definition}. Error: {err}')
                else:
                    accumulate_bad_lines.append(BadLine(d.definition_location.number, d.definition_location.line, err, d.definition_location.file))

    modes: KeyboardModeMap = {'': KeyboardMode()}

    for defn in defns:
        if defn.options.new_mode:
            modes[defn.options.new_mode] = nm = KeyboardMode(defn.options.new_mode)
            nm.on_unknown = defn.options.on_unknown
            nm.on_action = defn.options.on_action
            defn.definition = f'push_keyboard_mode {defn.options.new_mode}'
        try:
            m = modes[defn.options.mode]
        except KeyError:
            kerr = f'The keyboard mode {defn.options.mode} is unknown, ignoring the mapping'
            if accumulate_bad_lines is None:
                log_error(kerr)
            else:
                dl = defn.definition_location
                accumulate_bad_lines.append(BadLine(dl.number, dl.line, KeyError(kerr), dl.file))
            continue
        items = m.keymap[defn.trigger]
        if defn.is_sequence:
            items = m.keymap[defn.trigger] = [kd for kd in items if defn.rest != kd.rest or defn.options.when_focus_on != kd.options.when_focus_on]
        items.append(defn)
    opts.keyboard_modes = modes


def finalize_mouse_mappings(opts: Options, accumulate_bad_lines: Optional[List[BadLine]] = None) -> None:
    defns: List[MouseMapping] = []
    for d in opts.mouse_map:
        if d is None:  # clear_all_mouse_actions
            defns = []  # type: ignore
        else:
            try:
                defns.append(d.resolve_and_copy(opts.alatty_mod))
            except Exception as err:
                if accumulate_bad_lines is None:
                    log_error(f'Ignoring mouse_map with invalid action: {d.definition}. Error: {err}')
                else:
                    accumulate_bad_lines.append(BadLine(d.definition_location.number, d.definition_location.line, err, d.definition_location.file))
    mousemap: MouseMap = {}

    for defn in defns:
        if defn.definition:
            mousemap[defn.trigger] = defn.definition
        else:
            mousemap.pop(defn.trigger, None)
    opts.mousemap = mousemap


def parse_config(lines: Iterable[str], accumulate_bad_lines: Optional[List[BadLine]] = None) -> Dict[str, Any]:
    from .options.parse import create_result_dict, parse_conf_item
    ans: Dict[str, Any] = create_result_dict()
    parse_config_base(
        lines,
        parse_conf_item,
        ans,
        accumulate_bad_lines=accumulate_bad_lines
    )
    return ans


def load_config(*paths: str, overrides: Optional[Iterable[str]] = None, accumulate_bad_lines: Optional[List[BadLine]] = None) -> Options:
    from .options.parse import merge_result_dicts

    overrides = tuple(overrides) if overrides is not None else ()
    opts_dict, found_paths = _load_config(
        defaults, partial(parse_config, accumulate_bad_lines=accumulate_bad_lines), merge_result_dicts, *paths, overrides=overrides)
    opts = Options(opts_dict)

    opts.alias_map = build_action_aliases(opts.kitten_alias, 'kitten')
    opts.alias_map.update(build_action_aliases(opts.action_alias))
    finalize_keys(opts, accumulate_bad_lines)
    finalize_mouse_mappings(opts, accumulate_bad_lines)
    # delete no longer needed definitions, replacing with empty placeholders
    opts.kitten_alias = {}
    opts.action_alias = {}
    opts.mouse_map = []
    opts.map = []
    if opts.background_opacity < 1.0 and opts.macos_titlebar_color > 0:
        log_error('Cannot use both macos_titlebar_color and background_opacity')
        opts.macos_titlebar_color = 0
    opts.config_paths = found_paths
    opts.all_config_paths = paths
    opts.config_overrides = overrides
    return opts


class AlattyCommonOpts(TypedDict):
    select_by_word_characters: str


def common_opts_as_dict(opts: Optional[Options] = None) -> AlattyCommonOpts:
    if opts is None:
        opts = defaults
    return {
        'select_by_word_characters': opts.select_by_word_characters,
    }
