<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- in 3.0, current-group is initially absent -->
  
<xsl:template match="/">
<err>
	<xsl:value-of select="count(current-group())"/>
</err>
</xsl:template>
	
</xsl:transform>