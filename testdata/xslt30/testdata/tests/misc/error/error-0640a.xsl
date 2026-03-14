<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:key - circular key reference -->
<?spec xslt#errors?><?error XTDE0640?>


  <xsl:key name="key1" use="doc" match="key('key1', '12')"/>

  <xsl:template match="doc">
      <out>
         <xsl:value-of select="key('key1', '15')"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
