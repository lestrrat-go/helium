<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:output method="xml" version="1.1" />

  <xsl:template match="doc">
  	<xsl:param name="param" tunnel="yes" />
    <out>
		The best is: <xsl:value-of select="$param" />
	</out>
  </xsl:template>
  
</xsl:stylesheet> 