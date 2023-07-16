# package settings
# split version string into triple
# note: ${VERSION} is supposed to be set by parent
string(REPLACE "." ";" VERSION_LIST ${VERSION})
list(GET VERSION_LIST 0 CPACK_PACKAGE_VERSION_MAJOR)
list(GET VERSION_LIST 1 CPACK_PACKAGE_VERSION_MINOR)
list(GET VERSION_LIST 2 CPACK_PACKAGE_VERSION_PATCH)
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Whatsapp protocol plug-in for libpurple with pidgin")
set(CPACK_PACKAGE_VENDOR "Bryant Eadon")
set(CPACK_PACKAGE_DESCRIPTION "whatsmeow is a go-based whatsapp protocol plug-in for libpurple based on the whatsapp business apis.")
set(CPACK_PACKAGE_CONTACT "bryant@balancebitsconsulting.com")
# debian specific options
set(CPACK_GENERATOR "DEB" CACHE STRING "Which cpack generators to use.")
if (NOT ${CPACK_GENERATOR} STREQUAL "DEB")
    # CPACK_GENERATOR can be overridden on command-line
    message(WARNING "cpack genarator other than DEB has not been tested.")
endif()
# For now, we won't be doing any cross-compiling, so specifying amd64 here will cover most cases.
# If you have a different processor and need to cross-compile, say, ARM, then this won't build the package you need until it's modified.
set(CPACK_DEBIAN_PACKAGE_ARCHITECTURE "amd64")
set(CPACK_SOURCE_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}_${CPACK_DEBIAN_PACKAGE_ARCHITECTURE}")
set(CPACK_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}_${CPACK_DEBIAN_PACKAGE_ARCHITECTURE}")
set(CPACK_DEBIAN_PACKAGE_SHLIBDEPS ON)
set(CPACK_DEBIAN_PACKAGE_DEPENDS "libpurple0 (>= ${PURPLE_VERSION})") # TODO: libglib2.0-0 could be added here, but it is non-trivial to do. libpurple0 depends on it anyway, so we should be good.
if (${OPUSFILE_FOUND})
    set(CPACK_DEBIAN_PACKAGE_DEPENDS "${CPACK_DEBIAN_PACKAGE_DEPENDS}, libopusfile0 (>= ${OPUSFILE_VERSION})")
endif()
if ("${PIXBUF_FOUND}")
    set(CPACK_DEBIAN_PACKAGE_DEPENDS "${CPACK_DEBIAN_PACKAGE_DEPENDS}, libgdk-pixbuf-2.0-0 (>= ${PIXBUF_VERSION})")
endif()
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "Bryant Eadon") #required

include(CPack)
