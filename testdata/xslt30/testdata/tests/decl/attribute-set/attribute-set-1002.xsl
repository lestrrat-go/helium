<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
<!-- Purpose: Set attributes of an element created with xsl:element from single attribute set. -->
<!--  --><!-- ResultTree007 in NIST suite -->

<xsl:template match="/">
  <out>
    <xsl:element name="test1" use-attribute-sets="set1"/>
  </out>
</xsl:template>

<xsl:attribute-set name="set1">
  <xsl:attribute name="format">bold</xsl:attribute>
</xsl:attribute-set>

</xsl:stylesheet>
