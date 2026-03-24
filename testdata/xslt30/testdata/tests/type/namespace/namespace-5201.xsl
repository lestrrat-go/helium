<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-namespace-uri?>
<?spec fo#func-name?>
<?spec fo#func-local-name?>
  <!-- Purpose: Test of local-name and name() on namespace axis. -->
    <!-- The local-name() function should work on this axis, returning the same value as name().
     The XML parser has freedom to present namespaces in any order it wants.
     The input should have only one namespace if you want consistent results across parsers. -->

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="doc">
  <out>
    <xsl:value-of select="local-name(namespace::*[1])"/>
    <xsl:text>=</xsl:text>
    <xsl:value-of select="local-name(namespace::*[1])"/>
  </out>
</xsl:template>

</xsl:stylesheet>