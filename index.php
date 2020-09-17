<html>
<head>
        <meta charset="utf-8">
        <title>SleeperPy</title>
        <meta name="author" content="Will bollock">
        <link rel="stylesheet" href="tiers/style.css">
        <link rel="stylesheet" href="css/normalize.css">
         <link rel="stylesheet" href="css/skeleton.css">
         <link href="//fonts.googleapis.com/css?family=Raleway:400,300,600" rel="stylesheet" type="text/css">
         <meta name="viewport" content="width=device-width, initial-scale=1">
        <!-- https://codepen.io/alassetter/pen/cyrfB -->
</head>
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
    $filepath = "tiers/"."tiers_".$username.".html";
    #echo ("$filepath");
    $header = "Location: ".$filepath;
    header( "$header" );
    #echo file_get_contents("tiers.txt");

    # NOTE: tiers folder needs permissions for apache2
}
?>



<form action="" method="post">
<div class="container">
<div class="row">
<h1>SleeperPy</h1>

<ul>
    <li>Outputs your team's <a href="http://www.borischen.co/">Boris Chen</a> tiers across all Sleeper leagues.</li>
    <li><a href="https://github.com/wbollock/sleeperPy">GitHub Link</a></li>
    <li>It is best to run this on Thursday, as tiers are mostly updated by then.</li>
</ul>

Enter your Sleeper username: <input type="text" name="name">
<br>
<input type="submit" name="submit" value="Generate Tiers">

<br>
<br>


</form>
</div>
    </div>
  </div>




</body>
</html>