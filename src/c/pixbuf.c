#include "pixbuf.h"

#if __has_include("gdk-pixbuf/gdk-pixbuf.h")
// building with acces to pixbuf
#include <gdk-pixbuf/gdk-pixbuf.h>
static int
pixbuf_format_mimetype_comparator(GdkPixbufFormat *format, const char *type) {
    int cmp = 1;
    gchar **mime_types = gdk_pixbuf_format_get_mime_types(format);
    for (gchar **mime_type = mime_types; mime_type != NULL && *mime_type != NULL && cmp != 0; mime_type++) {
        cmp = g_strcmp0(type, *mime_type);
    }
    g_strfreev(mime_types);
    return cmp;
}

gboolean 
pixbuf_is_loadable_image_mimetype(const char *mimetype) {
    // check if mimetype is among the formats supported by pixbuf
    GSList *pixbuf_formats = gdk_pixbuf_get_formats();
    GSList *pixbuf_format = g_slist_find_custom(pixbuf_formats, mimetype, (GCompareFunc)pixbuf_format_mimetype_comparator);
    g_slist_free(pixbuf_formats);
    return pixbuf_format != NULL;
}
#else
// building without access to pixbuf
gboolean 
pixbuf_is_loadable_image_mimetype(const char *mimetype) {
    #pragma message "Warning: Building without GDK pixbuf loader detection."
    // blindly assume frontend can handle jpeg and png
    return g_strcmp0(mimetype, "image/jpeg") == 0 || g_strcmp0(mimetype, "image/png") == 0;
}
#endif
