<!DOCTYPE html>
<html lang="en">
    <head>
        <link rel="stylesheet" href="style.css">
        <!-- https://codepen.io/alassetter/pen/cyrfB -->
        <link rel="stylesheet" href="../css/normalize.css">
         <link rel="stylesheet" href="../css/skeleton.css">
         <link rel="stylesheet" href="../css/main.css">
         <link href="//fonts.googleapis.com/css?family=Raleway:400,300,600" rel="stylesheet" type="text/css">
         <meta name="viewport" content="width=device-width, initial-scale=1">
    </head>
    <?php
        # refresh button code
        # grab username from current URI, then call python script again
        $uri = $_SERVER['REQUEST_URI'];
        # sleeperPy/tiers/tiers_kiajon.html
        $m = array();
        preg_match('/tiers_(.*).php/', $uri, $m );
        # get username from current URI
        # returns string of username, so far so good
        $username = $m[1];

        if(isset($_POST['submit'])){
        
        //check if form was submitted
        chdir("..");
        $command = 'python3 sleeperPy.py '.$username;
        exec($command);
        $filepath = "tiers_".$username.".php";
        #echo ("$filepath");
        $header = "Location: ".$filepath;
        header( "$header" );
        #echo file_get_contents("tiers.txt");

        # NOTE: tiers folder needs permissions for apache2
    }
    ?>
<body>
    <!-- https://wboll.dev/SleeperPy-->
    <!-- https://github.com/wbollock/sleeperPy -->
    <div class="container">
        <div class="row">

    <h1><a href="https://wboll.dev/sleeperPy">SleeperPy</a></h1>
    <h3>Boris Chen Tiers for Sleeper Leagues</h3>
    
    <!-- <p>Notice: if you see "Craig Stevens", help me fight this bug by screenshoting your Sleeper FF roster and <a href="mailto:wbollock@gmail.com">emailing it to me</a>, along with your username.</p> -->
    <!-- <ul>
        <li><a href="https://wboll.dev/sleeperPy">sleeperPy Home</a></li>
    </ul> -->
    <!-- <center><form action="" method="post">
            <input type="submit" name="submit" value="Refresh Tiers" />
        </form></center> -->
        
            

        