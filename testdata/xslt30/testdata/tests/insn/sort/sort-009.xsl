<?xml version="1.0"?> 

<!-- A test for xsl:sort, multipart, alpha, some keys null -->
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="w3cgroup">
    <out>
      <xsl:apply-templates select="member">
        <xsl:sort select="alternate/name/first" order="ascending"/>
        <xsl:sort select="alternate/name/last" order="descending"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="member">
	    <xsl:value-of select="alternate/name/first"/><xsl:text> </xsl:text>
	    <xsl:value-of select="alternate/name/last"/><xsl:text>; </xsl:text>
  </xsl:template>
      
</xsl:stylesheet>