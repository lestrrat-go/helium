<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- test distinct-values on numeric values -->
<?spec fo#func-distinct-values?>
  
<xsl:template match="/">
<chapter>
    <xsl:value-of select="distinct-values((134, 1556, 12, 15, 134, 65, 12.0, 134e0))" separator=" "/>
</chapter>
</xsl:template>
	
</xsl:transform>