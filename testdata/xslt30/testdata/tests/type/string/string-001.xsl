<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#generate-id?>
<!-- PURPOSE:   Test of 'generate-id()'. -->
<xsl:template match="doc">
  <out>
    True: <xsl:value-of select="generate-id(a)=generate-id(a)"/>
    False: <xsl:value-of select="generate-id(a)=generate-id(b)"/>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
