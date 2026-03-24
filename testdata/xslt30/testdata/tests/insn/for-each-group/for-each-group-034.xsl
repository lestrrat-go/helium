<?spec xslt#grouping?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- test that distinct-values treats NaNs as equal -->
  <?category distinct-values?>
  
<xsl:template match="/">
<chapter>
    <xsl:value-of select="distinct-values((1, 2, 4, number('NaN'), 2, 6, 4, number('nan')))" separator=" "/>
</chapter>
</xsl:template>
	
</xsl:transform>