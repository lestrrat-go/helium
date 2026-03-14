<?xml version="1.0"?> 

<!-- test for id() applied to an empty sequence -->
<?spec fo#func-id?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:template match="/">
      <P><xsl:value-of select="name(id(()))"/></P>
  </xsl:template>
 
</xsl:stylesheet>
