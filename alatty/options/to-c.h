/*
 * Copyright (C) 2021 Kovid Goyal <kovid at kovidgoyal.net>
 *
 * Distributed under terms of the GPL3 license.
 */

#pragma once

#include "../state.h"
#include "../colors.h"

static inline float
PyFloat_AsFloat(PyObject *o) {
    return (float)PyFloat_AsDouble(o);
}

static inline color_type
color_as_int(PyObject *color) {
    if (!PyObject_TypeCheck(color, &Color_Type)) { PyErr_SetString(PyExc_TypeError, "Not a Color object"); return 0; }
    Color *c = (Color*)color;
    return c->color.val & 0xffffff;
}

static inline color_type
color_or_none_as_int(PyObject *color) {
    if (color == Py_None) return 0;
    return color_as_int(color);
}

static inline color_type
active_border_color(PyObject *color) {
    if (color == Py_None) return 0x00ff00;
    return color_as_int(color);
}


static inline monotonic_t
parse_s_double_to_monotonic_t(PyObject *val) {
    return s_double_to_monotonic_t(PyFloat_AsDouble(val));
}

static inline monotonic_t
parse_ms_long_to_monotonic_t(PyObject *val) {
    return ms_to_monotonic_t(PyLong_AsUnsignedLong(val));
}

#define STR_SETTER(name) { \
    free(opts->name); opts->name = NULL; \
    if (src == Py_None || !PyUnicode_Check(src)) return; \
    Py_ssize_t sz; \
    const char *s = PyUnicode_AsUTF8AndSize(src, &sz); \
    opts->name = calloc(sz + 1, sizeof(s[0])); \
    if (opts->name) memcpy(opts->name, s, sz); \
}

static void
window_logo_path(PyObject *src, Options *opts) { STR_SETTER(default_window_logo); }

#undef STR_SETTER

static void
parse_font_mod_size(PyObject *val, float *sz, AdjustmentUnit *unit) {
    PyObject *mv = PyObject_GetAttrString(val, "mod_value");
    if (mv) {
        *sz = PyFloat_AsFloat(PyTuple_GET_ITEM(mv, 0));
        long u = PyLong_AsLong(PyTuple_GET_ITEM(mv, 1));
        switch (u) { case POINT: case PERCENT: case PIXEL: *unit = u; break; }
    }
}

static void
modify_font(PyObject *mf, Options *opts) {
#define S(which) { PyObject *v = PyDict_GetItemString(mf, #which); if (v) parse_font_mod_size(v, &opts->which.val, &opts->which.unit); }
    S(underline_position); S(underline_thickness); S(strikethrough_thickness); S(strikethrough_position);
    S(cell_height); S(cell_width); S(baseline);
#undef S
}

static int
macos_colorspace(PyObject *csname) {
    if (PyUnicode_CompareWithASCIIString(csname, "srgb") == 0) return 1;
    if (PyUnicode_CompareWithASCIIString(csname, "displayp3") == 0) return 2;
    return 0;
}

static inline void
free_menu_map(Options *opts) {
    if (opts->global_menu.entries) {
        for (size_t i=0; i < opts->global_menu.count; i++) {
            struct MenuItem *e = opts->global_menu.entries + i;
            if (e->definition) { free((void*)e->definition); }
            if (e->location) {
                for (size_t l=0; l < e->location_count; l++) { free((void*)e->location[l]); }
                free(e->location);
            }
        }
        free(opts->global_menu.entries); opts->global_menu.entries = NULL;
    }
    opts->global_menu.count = 0;
}

static void
menu_map(PyObject *entry_dict, Options *opts) {
    if (!PyDict_Check(entry_dict)) { PyErr_SetString(PyExc_TypeError, "menu_map entries must be a dict"); return; }
    free_menu_map(opts);
    size_t maxnum = PyDict_Size(entry_dict);
    opts->global_menu.count = 0;
    opts->global_menu.entries = calloc(maxnum, sizeof(opts->global_menu.entries[0]));
    if (!opts->global_menu.entries) { PyErr_NoMemory(); return; }

    PyObject *key, *value;
    Py_ssize_t pos = 0;

    while (PyDict_Next(entry_dict, &pos, &key, &value)) {
        if (PyTuple_Check(key) && PyTuple_GET_SIZE(key) > 1 && PyUnicode_Check(value) && PyUnicode_CompareWithASCIIString(PyTuple_GET_ITEM(key, 0), "global") == 0) {
            struct MenuItem *e = opts->global_menu.entries + opts->global_menu.count++;
            e->location_count = PyTuple_GET_SIZE(key) - 1;
            e->location = calloc(e->location_count, sizeof(e->location[0]));
            if (!e->location) { PyErr_NoMemory(); return; }
            e->definition = strdup(PyUnicode_AsUTF8(value));
            if (!e->definition) { PyErr_NoMemory(); return; }
            for (size_t i = 0; i < e->location_count; i++) {
                e->location[i] = strdup(PyUnicode_AsUTF8(PyTuple_GET_ITEM(key, i+1)));
                if (!e->location[i]) { PyErr_NoMemory(); return; }
            }
        }
    }
}

static void
text_composition_strategy(PyObject *val, Options *opts) {
    if (!PyUnicode_Check(val)) { PyErr_SetString(PyExc_TypeError, "text_rendering_strategy must be a string"); return; }
    opts->text_old_gamma = false;
    opts->text_gamma_adjustment = 1.0f; opts->text_contrast = 0.f;
    if (PyUnicode_CompareWithASCIIString(val, "platform") == 0) {
#ifdef __APPLE__
        opts->text_gamma_adjustment = 1.7f; opts->text_contrast = 30.f;
#endif
    }
    else if (PyUnicode_CompareWithASCIIString(val, "legacy") == 0) {
        opts->text_old_gamma = true;
    } else {
        RAII_PyObject(parts, PyUnicode_Split(val, NULL, 2));
        int size = PyList_GET_SIZE(parts);
        if (size < 1 || 2 < size) { PyErr_SetString(PyExc_ValueError, "text_rendering_strategy must be of the form number:[number]"); return; }

        if (size > 0) {
            RAII_PyObject(ga, PyFloat_FromString(PyList_GET_ITEM(parts, 0)));
            if (PyErr_Occurred()) return;
            opts->text_gamma_adjustment = MAX(0.01f, PyFloat_AsFloat(ga));
        }

        if (size > 1) {
            RAII_PyObject(contrast, PyFloat_FromString(PyList_GET_ITEM(parts, 1)));
            if (PyErr_Occurred()) return;
            opts->text_contrast = MAX(0.0f, PyFloat_AsFloat(contrast));
            opts->text_contrast = MIN(100.0f, opts->text_contrast);
        }
    }
}

static char_type*
list_of_chars(PyObject *chars) {
    if (!PyUnicode_Check(chars)) { PyErr_SetString(PyExc_TypeError, "list_of_chars must be a string"); return NULL; }
    char_type *ans = calloc(PyUnicode_GET_LENGTH(chars) + 1, sizeof(char_type));
    if (ans) {
        for (ssize_t i = 0; i < PyUnicode_GET_LENGTH(chars); i++) {
            ans[i] = PyUnicode_READ(PyUnicode_KIND(chars), PyUnicode_DATA(chars), i);
        }
    }
    return ans;
}

static void
select_by_word_characters(PyObject *chars, Options *opts) {
    free(opts->select_by_word_characters);
    opts->select_by_word_characters = list_of_chars(chars);
}

static void
select_by_word_characters_forward(PyObject *chars, Options *opts) {
    free(opts->select_by_word_characters_forward);
    opts->select_by_word_characters_forward = list_of_chars(chars);
}

static void
tab_bar_style(PyObject *val, Options *opts) {
    opts->tab_bar_hidden = PyUnicode_CompareWithASCIIString(val, "hidden") == 0 ? true: false;
}

static void
tab_bar_margin_height(PyObject *val, Options *opts) {
    if (!PyTuple_Check(val) || PyTuple_GET_SIZE(val) != 2) {
        PyErr_SetString(PyExc_TypeError, "tab_bar_margin_height is not a 2-item tuple");
        return;
    }
    opts->tab_bar_margin_height.outer = PyFloat_AsDouble(PyTuple_GET_ITEM(val, 0));
    opts->tab_bar_margin_height.inner = PyFloat_AsDouble(PyTuple_GET_ITEM(val, 1));
}

static void
resize_debounce_time(PyObject *src, Options *opts) {
    opts->resize_debounce_time.on_end = s_double_to_monotonic_t(PyFloat_AsDouble(PyTuple_GET_ITEM(src, 0)));
    opts->resize_debounce_time.on_pause = s_double_to_monotonic_t(PyFloat_AsDouble(PyTuple_GET_ITEM(src, 1)));
}

static void
free_allocs_in_options(Options *opts) {
    free_menu_map(opts);
#define F(x) free(opts->x); opts->x = NULL;
    F(select_by_word_characters); F(select_by_word_characters_forward);
    F(default_window_logo);
#undef F
}
