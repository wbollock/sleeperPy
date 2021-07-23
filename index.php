<html>
<head>
        <meta charset="UTF-8">
        <title>SleeperPy</title>
        <meta name="author" content="Will bollock">
        <link rel="stylesheet" href="tiers/style.css">
        <!-- https://codepen.io/alassetter/pen/cyrfB -->
        <!-- credit for skeleton http://getskeleton.com/ -->
        <link rel="stylesheet" href="css/normalize.css">
         <link rel="stylesheet" href="css/skeleton.css">
         <link rel="stylesheet" href="css/main.css">
         <link href="//fonts.googleapis.com/css?family=Raleway:400,300,600" rel="stylesheet" type="text/css">
         <meta name="viewport" content="width=device-width, initial-scale=1">
         <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.1/jquery.min.js"></script>
         <script>
             // https://stackoverflow.com/questions/38217274/loading-gif-on-normal-form-submit
                $(document).ready(function(){
        $("#userform").on("submit", function(){
            $("#pageloader").fadeIn();
        });//submit
        });//document ready
        </script>
        
</head>
<body>



<?php
$username = "";
if(isset($_POST['submit'])){ //check if form was submitted

    $username = htmlspecialchars($_POST['name']);
    $command = 'python3 sleeperPy.py '.$username;
    #$script = escapeshellcmd('python3 sleeperPy.py ').$username;
    #$output = shell_exec($command);
    #readfile("tiers.txt");
    exec($command);
    $filepath = "tiers/"."tiers_".$username.".php";
    #echo ("$filepath");
    $header = "Location: ".$filepath;
    header( "$header" );
    
    #echo file_get_contents("tiers.txt");

    # NOTE: tiers folder needs permissions for apache2
}
?>



<form action="" method="post" id="userform">
<div class="container">
<div class="row centerinput">
<!-- <div class="eight columns"> -->
<h1 class="homepageHeader"><a href="https://wboll.dev/sleeperPy">SleeperPy</a></h1>

<!-- <ul>
    <li>Outputs your team's <a href="http://www.borischen.co/">Boris Chen</a> tiers across all Sleeper leagues.</li>
    <li><a href="https://github.com/wbollock/sleeperPy">GitHub Link</a></li>
    <li>It is best to run this on Wednesday or Thursday, as tiers are mostly updated by then.</li>
</ul> -->

<!-- <h4 class="homepageHeader"><b>NOTICE: The 2020 Fantasy Football season is over. Please check back in 2021!</b></h4> -->
<h5 id="infoText" style="text-align:left;">Displays your team's <a href="http://www.borischen.co/">Boris Chen</a> tiers across all Sleeper leagues.</h5>

<input id="inputButton" type="text" name="name" required placeholder="Sleeper Username" pattern="^\S+$"
oninvalid="this.setCustomValidity('Username without spaces')"
    oninput="this.setCustomValidity('')" >
<br>
<input id="generateTiers" type="submit" name="submit" value="Generate Tiers">

<br>
<br>



</form>


    </div>
    <!-- <div id="pageloader">
   <img src="loading2.gif" alt="processing..." />
</div> -->
</div>
</div>

<div class="container">
<div class="row centerinput">

<footer>
<ul>
    <li>In the "Tiers" column, lower is better.</li>
    <!-- <li>It is best to run this on Wednesday or Thursday, as tiers are mostly updated by then.</li> -->
    <li><a href="https://github.com/wbollock/sleeperPy">GitHub Repo</a> | 
    <a href="http://www.borischen.co/">Source of Tiers</a> | 
    <a href="https://codepen.io/alassetter/pen/cyrfB">CSS Table Styling</a> | 
    <a href="http://getskeleton.com/">General Styling</a>
     </li>
    <!-- <li><a href="http://www.borischen.co/">Source of Tiers </a></li>
    <li><a href="https://codepen.io/alassetter/pen/cyrfB">CSS Table Styling</a></li>
    <li><a href="http://getskeleton.com/">General Styling</a></li> -->
</ul>
</footer>

</div>
</div>

  
</body>
</html>
