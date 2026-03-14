<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: idkey23 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test for id() with a non-matching value. -->

<xsl:template match="/">
  <P><xsl:value-of select="id('nomatch')/@id"/></P>
</xsl:template>

</xsl:stylesheet>
