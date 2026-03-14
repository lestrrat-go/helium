<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test sequence expressions -->
<?spec xpath#construct_seq?>

<xsl:variable name="e1" select="()"/>
<xsl:variable name="e2" select="(1)"/>
<xsl:variable name="e3" select="(10,20)"/>
<xsl:variable name="e4" select="(100,200,300)"/>
<xsl:variable name="e5" select="($e1,$e2,$e3,$e4)"/>
<xsl:variable name="e6" select="20 to 29"/>
<xsl:variable name="e7" select="reverse(20 to 29)"/>

<xsl:template match="/">
  <out>;
    <xsl:value-of select="count($e1)"/>;
    <xsl:value-of select="count($e2)"/>;
    <xsl:value-of select="count($e3)"/>;
    <xsl:value-of select="count($e4)"/>;
    <xsl:value-of select="count($e5)"/>;
    <xsl:value-of select="count($e6)"/>;
    <xsl:value-of select="count($e7)"/>;                     
  </out>
</xsl:template>
 
</xsl:stylesheet>
