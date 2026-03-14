<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <!-- FileName: copy2116 -->
    <!-- Purpose: Use copy-of() to copy an attribute; and check it uses validation=preserve -->


<xsl:template match="/">
  <xsl:variable name="att" as="attribute()">
    <xsl:attribute name="a" select="'2010-01-01'" type="xs:date"/>
  </xsl:variable>  
  <out>
    <xsl:sequence select="format-date(copy-of($att), '[D1] [MN] [Y]')"/>
  </out>
</xsl:template>

</xsl:stylesheet>