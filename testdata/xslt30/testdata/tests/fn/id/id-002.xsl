<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-id?>
<!-- test id() function with a node-set argument and multiple IDs -->

  <xsl:template match="/">
    <list new="xslt2.0"><xsl:value-of select="id(//b/@ref)/@id" separator=","/></list>
  </xsl:template>
 
</xsl:stylesheet>
