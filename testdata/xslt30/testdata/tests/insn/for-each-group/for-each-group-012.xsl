<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- test distinct-values on string values -->
<?spec fo#func-distinct-values?>
  
<xsl:template match="/">
<chapter>
    <xsl:value-of select="distinct-values(/cities/city/@country)" separator=" "/>
</chapter>
</xsl:template>
	
</xsl:transform>