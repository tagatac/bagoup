%module pdf
%{
#define BUILDING_WKHTMLTOX
#include "pdf.h"
%}

struct wkhtmltopdf_global_settings;
typedef struct wkhtmltopdf_global_settings wkhtmltopdf_global_settings;

struct wkhtmltopdf_object_settings;
typedef struct wkhtmltopdf_object_settings wkhtmltopdf_object_settings;

struct wkhtmltopdf_converter;
typedef struct wkhtmltopdf_converter wkhtmltopdf_converter;

typedef void (*wkhtmltopdf_str_callback)(wkhtmltopdf_converter * converter, const char * str);
typedef void (*wkhtmltopdf_int_callback)(wkhtmltopdf_converter * converter, const int val);
typedef void (*wkhtmltopdf_void_callback)(wkhtmltopdf_converter * converter);

int wkhtmltopdf_init(int use_graphics);
int wkhtmltopdf_deinit();
int wkhtmltopdf_extended_qt();
const char * wkhtmltopdf_version();

wkhtmltopdf_global_settings * wkhtmltopdf_create_global_settings();
void wkhtmltopdf_destroy_global_settings(wkhtmltopdf_global_settings *);

wkhtmltopdf_object_settings * wkhtmltopdf_create_object_settings();
void wkhtmltopdf_destroy_object_settings(wkhtmltopdf_object_settings *);

int wkhtmltopdf_set_global_setting(wkhtmltopdf_global_settings * settings, const char * name, const char * value);
int wkhtmltopdf_get_global_setting(wkhtmltopdf_global_settings * settings, const char * name, char * value, int vs);
int wkhtmltopdf_set_object_setting(wkhtmltopdf_object_settings * settings, const char * name, const char * value);
int wkhtmltopdf_get_object_setting(wkhtmltopdf_object_settings * settings, const char * name, char * value, int vs);


wkhtmltopdf_converter * wkhtmltopdf_create_converter(wkhtmltopdf_global_settings * settings);
void wkhtmltopdf_destroy_converter(wkhtmltopdf_converter * converter);

void wkhtmltopdf_set_debug_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_str_callback cb);
void wkhtmltopdf_set_info_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_str_callback cb);
void wkhtmltopdf_set_warning_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_str_callback cb);
void wkhtmltopdf_set_error_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_str_callback cb);
void wkhtmltopdf_set_phase_changed_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_void_callback cb);
void wkhtmltopdf_set_progress_changed_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_int_callback cb);
void wkhtmltopdf_set_finished_callback(wkhtmltopdf_converter * converter, wkhtmltopdf_int_callback cb);


int wkhtmltopdf_convert(wkhtmltopdf_converter * converter);
void wkhtmltopdf_add_object(
 wkhtmltopdf_converter * converter, wkhtmltopdf_object_settings * setting, const char * data);

int wkhtmltopdf_current_phase(wkhtmltopdf_converter * converter);
int wkhtmltopdf_phase_count(wkhtmltopdf_converter * converter);
const char * wkhtmltopdf_phase_description(wkhtmltopdf_converter * converter, int phase);
const char * wkhtmltopdf_progress_string(wkhtmltopdf_converter * converter);
int wkhtmltopdf_http_error_code(wkhtmltopdf_converter * converter);
long wkhtmltopdf_get_output(wkhtmltopdf_converter * converter, const unsigned char **);
