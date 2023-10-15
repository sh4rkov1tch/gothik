package main

/* Very simple things going on here */
func main() {
	token := discord_get_token()
	bot, command := discord_init_bot(token)
	discord_run_bot(bot, command)
}
