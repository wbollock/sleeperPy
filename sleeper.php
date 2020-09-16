<html>
<body>


<form action="#" method="post">
Sleeper Username: <input type="text" name="name"><br>
<input type="submit" name="submit" value="submit">
</form>

<?php
$username = $_POST['name'];
$script = escapeshellcmd('python3 sleeper.Py ').$username;
$output = shell_exec($command);
echo $output;

?>

</body>
</html>