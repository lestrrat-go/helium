<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test "xsl:for-each" over a sequence -->
<?spec xslt#for-each?>

<xsl:template match="/">
  <out>
    <xsl:for-each select="1 to 10">
      <a><xsl:value-of select=".+1"/></a>
    </xsl:for-each>
  </out>
</xsl:template>
 
</xsl:stylesheet>
