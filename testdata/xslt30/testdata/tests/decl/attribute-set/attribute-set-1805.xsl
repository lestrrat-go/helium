<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0"
    xmlns:ap="http://attribco.com/"
    exclude-result-prefixes="ap">

 <?spec xslt#attribute-sets?>
    <!-- Purpose: Test attribute set with a qualified name. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:attribute-set name="ap:set">
  <xsl:attribute name="color">black</xsl:attribute>
</xsl:attribute-set>

<xsl:template match="/">
  <out>
    <test1 xsl:use-attribute-sets="ap:set"></test1>
  </out>
</xsl:template>

</xsl:stylesheet>