<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- in 3.0, current-grouping-key is initially absent -->
  
<xsl:template match="/">
<err>
	<xsl:value-of select="current-grouping-key()"/>
</err>
</xsl:template>
	
</xsl:transform>