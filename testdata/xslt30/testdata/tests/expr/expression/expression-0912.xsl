<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test sorted "xsl:for-each" over a sequence -->
<?spec xslt#for-each?>

<xsl:template match="/">
  <out>
    <xsl:for-each select="(12, 5, 14, -2, 19, -100, 0, 14)">
      <xsl:sort data-type="number"/>
      <xsl:value-of select="."/><xsl:text>;</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>
 
</xsl:stylesheet>
