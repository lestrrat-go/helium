<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test position() and last() with sequence expressions -->
<?spec fo#func-position?>
<?spec fo#func-last?>

<xsl:variable name="e1" select="()"/>
<xsl:variable name="e2" select="(1)"/>
<xsl:variable name="e3" select="(10,20)"/>
<xsl:variable name="e4" select="(100,200,300)"/>
<xsl:variable name="e5" select="($e1,$e2,$e3,$e4)"/>
<xsl:variable name="e6" select="20 to 29"/>
<xsl:variable name="e7" select="reverse(20 to 29)"/>

<xsl:template match="/">
  <out>;
    (<xsl:for-each select="()"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="(1)"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="(10,20)"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="(1, 2, 5 to 9)"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="($e3, $e4)"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="($e6, $e7)"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
    (<xsl:for-each select="1 to 9"><xsl:value-of select="concat(position(), ' of ', last())"/>; </xsl:for-each>);
  </out>
</xsl:template>
 
</xsl:stylesheet>
