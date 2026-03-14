<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string-length?>
<!-- PURPOSE:   Test optimization of (string-length = 0) . -->

<xsl:template match="doc">
<out>
    <z><xsl:value-of select="for $x in a return string-length($x)=0" separator=","/></z>;
    <z><xsl:value-of select="for $x in a return string-length($x)&gt;0" separator=","/></z>;   
    <z><xsl:value-of select="for $x in a return string-length($x)!=0" separator=","/></z>; 
    <z><xsl:value-of select="for $x in a return string-length($x)=1" separator=","/></z>;            
</out>
</xsl:template>
</xsl:stylesheet>