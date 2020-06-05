/*
 * This is a boiled-down copy of libpurple's proxy.h just with the structs for CGO to understand.
 * 
 * This is not a real header. It must not be included before the real one.
 */
#ifndef _PURPLE_PROXY_H_
#define _PURPLE_PROXY_H_
typedef enum
{
	PURPLE_PROXY_USE_GLOBAL = -1,  /**< Use the global proxy information. */
	PURPLE_PROXY_NONE = 0,         /**< No proxy.                         */
	PURPLE_PROXY_HTTP,             /**< HTTP proxy.                       */
	PURPLE_PROXY_SOCKS4,           /**< SOCKS 4 proxy.                    */
	PURPLE_PROXY_SOCKS5,           /**< SOCKS 5 proxy.                    */
	PURPLE_PROXY_USE_ENVVAR,       /**< Use environmental settings.       */
	PURPLE_PROXY_TOR               /**< Use a Tor proxy (SOCKS 5 really)  */

} PurpleProxyType;

typedef struct
{
	PurpleProxyType type;
	char *host;
	int   port;
	char *username;
	char *password;
} PurpleProxyInfo;
#endif /* _PURPLE_PROXY_H_ */
