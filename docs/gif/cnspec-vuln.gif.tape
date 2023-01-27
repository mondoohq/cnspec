Output cnspec-vuln.gif
Set Padding 20
Set Shell bash
Set FontSize 13
Set Width 850
Set Height 580
Hide
Type "clear"
Enter
Sleep 1s
Show
Type "cnspec vuln local"
Sleep 1s
Enter
Sleep 10s
Hide
Type "clear"
Enter
Sleep 1s
Show
Type "cnspec vuln ssh tsmith@172.16.1.50 --identity-file ~/.ssh/id_rsa"
Sleep 1s
Enter
Sleep 12s
Hide
Type "clear"
Enter
Sleep 1s
Show
Type "cnspec vuln docker image ubuntu:22.04"
Sleep 1s
Enter
Sleep 10
