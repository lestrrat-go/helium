<?xml version="1.0" encoding="iso-8859-1"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" >

<!-- Error: xsl:decimal-format  -->
<?spec xslt#errors?><?error XTDE1280?>

<xsl:decimal-format decimal-separator="."/>

  <xsl:template match="doc" name="this">
    <out>
      <xsl:value-of select="format-number(1234.56, '####0.00', 'undefined')"/>
      <xsl:message>Error not detected!</xsl:message>
    </out>
  </xsl:template>
  

</xsl:stylesheet>