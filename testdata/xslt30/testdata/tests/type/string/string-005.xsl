<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string?>
<!-- PURPOSE:   Test of 'string()' conversion function. -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="string(doc)"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
