<?xml version="1.0"?> 
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"> 
<xsl:output method="xml"/> 

<?spec xpath#id-predicates?>
<!-- Bug reported as https://sourceforge.net/forum/forum.php?thread_id=637041&forum_id=94027 -->
<!-- Apparent cause is an error in the optimization of constant [true()] predicates -->

<xsl:template match="a"> 
<out>
Test 1a: b [true()] 
<xsl:for-each select="b [true()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 1b: b [true()] [last()] 
<xsl:for-each select="b [true()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 2a: b [true()] [4 > position()] 
<xsl:for-each select="b [true()] [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 2b: b [true()] [4 > position()] [last()] 
<xsl:for-each select="b [true()] [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 3a: b [4 > position()] 
<xsl:for-each select="b [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 3b: b [4 > position()] [last()] 
<xsl:for-each select="b [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 4a: b [4 > position()] [true()] 
<xsl:for-each select="b [4 > position()] [true()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 4b: b [4 > position()] [true()] [last()] 
<xsl:for-each select="b [4 > position()] [true()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 5a: b [4 > position()] [4 > position()] 
<xsl:for-each select="b [4 > position()] [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 5b: b [4 > position()] [4 > position()] [last()] 
<xsl:for-each select="b [4 > position()] [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 6a: b [true()] [4 > position()] [4 > position()] 
<xsl:for-each select="b [true()] [4 > position()] [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 6b: b [true()] [4 > position()] [4 > position()] [last()] 
<xsl:for-each select="b [true()] [4 > position()] [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 7a: b [4 > position()] [true()] [4 > position()] 
<xsl:for-each select="b [4 > position()] [true()] [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 7b: b [4 > position()] [true()] [4 > position()] [last()] 
<xsl:for-each select="b [4 > position()] [true()] [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 8a: b [position() > 1] [4 > position()] 
<xsl:for-each select="b [position() > 1] [4 > position()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 

Test 8b: b [position() > 1] [4 > position()] [last()] 
<xsl:for-each select="b [position() > 1] [4 > position()] [last()]"> 
<xsl:value-of select="."/> 
</xsl:for-each> 
</out>
</xsl:template> 
</xsl:stylesheet> 

<!--
########## The output with saxon 6.4.3 or saxon 6.5: 

Test 1a: b [true()] 
01234 

Test 1b: b [true()] [last()] 
4 

Test 2a: b [true()] [4 > position()] 
012 

Test 2b: b [true()] [4 > position()] [last()] 
4 

Test 3a: b [4 > position()] 
012 

Test 3b: b [4 > position()] [last()] 
2 

Test 4a: b [4 > position()] [true()] 
012 

Test 4b: b [4 > position()] [true()] [last()] 
2 

Test 5a: b [4 > position()] [4 > position()] 
012 

Test 5b: b [4 > position()] [4 > position()] [last()] 
2 

Test 6a: b [true()] [4 > position()] [4 > position()] 
012 

Test 6b: b [true()] [4 > position()] [4 > position()] [last()] 
4 

Test 7a: b [4 > position()] [true()] [4 > position()] 
012 

Test 7b: b [4 > position()] [true()] [4 > position()] [last()] 
2 

Test 8a: b [position() > 1] [4 > position()] 
123 

Test 8b: b [position() > 1] [4 > position()] [last()] 
4 
-->
