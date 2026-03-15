<?xml version="1.0"?> 

<!-- A test for xsl:sort, multi-part sort keys -->
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="w3cgroup">
    <out>
      <xsl:apply-templates select="member">
        <xsl:sort select="primary/name/first" order="descending"/>
        <xsl:sort select="primary/name/last" order="ascending"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="member">
	    <xsl:value-of select="primary/name/first"/><xsl:text> </xsl:text>
	    <xsl:value-of select="primary/name/last"/><xsl:text>; </xsl:text>
  </xsl:template>
      
</xsl:stylesheet>