<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
    <!-- Purpose: Test use of leading underscore in names. -->

<xsl:attribute-set name="_a_set">
  <xsl:attribute name="_a_color">black</xsl:attribute>
</xsl:attribute-set>

<xsl:template match="/">
  <out>
    <test xsl:use-attribute-sets="_a_set" />
  </out>
</xsl:template>

</xsl:stylesheet>