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
    $filepath = "tiers/"."tiers_".$username.".txt";
    #echo ("$filepath");
    $header = "Location: ".$filepath;
    header( "$header" );
    #echo file_get_contents("tiers.txt");

    # NOTE: tiers folder needs permissions for apache2
}
?>



<form action="" method="post">
<h1>SleeperPy</h1>
<p>Outputs your team's <a href="http://www.borischen.co/">Boris Chen</a> tiers across all Sleeper leagues. </p>
<a href="https://github.com/wbollock/sleeperPy">GitHub Link</a>
<br>
<br>
Enter your Sleeper username: <input type="text" name="name">
<br>
<br>
<input type="submit" name="submit" value="Generate Tiers">

<br>
<br>
<p>It is best to run this on Thursday, as tiers are mostly updated by then.</p>

</form>




</body>
</html>