<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position76 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() with namespace axis. -->
  <!-- The XML parser has freedom to present namespaces in any order it wants.
     Nevertheless, the position() function should work on this axis, not raise an error.
     The input should have only one namespace if you want consistent results across parsers. -->

<xsl:template match="doc">
  <out><xsl:value-of select="name(namespace::*[1])"/></out>
</xsl:template>

</xsl:stylesheet>