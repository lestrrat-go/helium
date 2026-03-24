<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test xsl:value-of with sequence expressions -->
<?spec xslt#value-of?>

<xsl:variable name="e1" select="()"/>
<xsl:variable name="e2" select="(1)"/>
<xsl:variable name="e3" select="(10,20)"/>
<xsl:variable name="e4" select="(100,200,300)"/>
<xsl:variable name="e5" select="($e1,$e2,$e3,$e4)"/>
<xsl:variable name="e6" select="20 to $limit"/>
<xsl:variable name="e7" select="reverse(20 to 29)"/>

<xsl:param name="limit" select="29"/>

<xsl:template match="/">
  <out>;
    (<xsl:value-of select="()" separator=","/>)
    (<xsl:value-of select="(1)" separator=","/>)
    (<xsl:value-of select="(10,20)" separator=","/>)
    (<xsl:value-of select="(1, 2, 5 to 9)" separator=","/>)
    (<xsl:value-of select="($e3, $e4)" separator=","/>)
    (<xsl:value-of select="($e6, $e7)" separator=","/>)
    (<xsl:value-of select="1 to 9" separator=""/>)
  </out>
</xsl:template>
 
</xsl:stylesheet>
