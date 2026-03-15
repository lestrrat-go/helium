<?xml version="1.0"?> 

<!-- A test for xsl:sort, multipart keys, alpha -->
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="w3cgroup">
    <out>
      <xsl:apply-templates select="member">
        <xsl:sort select="primary/name/last"/>
        <xsl:sort select="primary/name/first"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="member">
	    <xsl:value-of select="primary/name/last"/><xsl:text> </xsl:text>
	    <xsl:value-of select="primary/name/first"/><xsl:text>; </xsl:text>
  </xsl:template>
      
</xsl:stylesheet>