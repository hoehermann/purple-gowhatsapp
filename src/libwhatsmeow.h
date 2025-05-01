#pragma message "Warning: Building with a dummy header. Please execute `go build` again."

extern void gowhatsapp_go_login(PurpleAccount* account, char* purple_user_dir, char* username, char* password, char* proxy_uri);
extern void gowhatsapp_go_close(PurpleAccount* account);
extern void gowhatsapp_go_logout(PurpleAccount* account);
extern int gowhatsapp_go_send_message(PurpleAccount* account, char* who, char* message, int is_group);
extern char* gowhatsapp_go_send_file(PurpleAccount* account, char* who, char* filename);
extern void gowhatsapp_go_mark_read_conversation(PurpleAccount* account, char* who);
extern void gowhatsapp_go_send_presence(PurpleAccount* account, char* presence);
extern void gowhatsapp_go_subscribe_presence(PurpleAccount* account, char* who);
extern char** gowhatsapp_go_query_group_participants(PurpleAccount* account, char* groupid);
extern void gowhatsapp_go_query_groups(PurpleAccount* account);
extern void gowhatsapp_go_get_contacts(PurpleAccount* account);
extern void gowhatsapp_go_request_profile_picture(PurpleAccount* account, char* who, char* picture_date, char* picture_id);
