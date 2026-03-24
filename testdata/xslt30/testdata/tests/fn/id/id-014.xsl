<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">

<?spec xslt#copy-of?>
<!-- AElfred parser bug - parameter entities in internal subset -->
<!-- (see XML source document) -->
  
  <xsl:strip-space elements="*"/>
  <xsl:template match="/">
    <xsl:copy-of select="*"/>
  </xsl:template>

</xsl:transform>
