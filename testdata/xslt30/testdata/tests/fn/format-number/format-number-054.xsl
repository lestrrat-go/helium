<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: Test use of multiple non-conflicting unnamed decimal formats. 
        See Saxon bug http://dev.saxonica.com/community/issues/1571 -->

<xsl:decimal-format decimal-separator=","/>
<xsl:decimal-format grouping-separator="."/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(3931.4857,'#.###,000')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
