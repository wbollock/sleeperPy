<html>
<body>

<?php
$username = "";
if(isset($_POST['submit'])){ //check if form was submitted
    $username = $_POST['name'];
    $command = 'python3 sleeperPy.py '.$username;
    #$script = escapeshellcmd('python3 sleeperPy.py ').$username;
    #$output = shell_exec($command);
    #readfile("tiers.txt");
    exec($command);
    header( "Location: tiers.txt" );
    #echo file_get_contents("tiers.txt");
}
?>



<form action="" method="post">
Sleeper Username: <input type="text" name="name"><br>
<input type="submit" name="submit" value="submit">
</form>




</body>
</html>