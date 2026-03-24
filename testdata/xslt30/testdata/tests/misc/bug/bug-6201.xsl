<?xml version="1.0" encoding="UTF-8"?>
<stylesheet version="1.0" xmlns="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#current-function?>
<!-- optimization bug in 7.6.5, -->

<template match="/">
    <variable name="a" select="*[. + current()=1]"/>
    <element name="x" namespace=""><value-of select="$a"/></element> 
</template>

</stylesheet>
