<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
      xmlns:axsl="http://www.w3.org/1999/XSL/TransAlias">

<xsl:output method="xml" indent="no"/>

<?spec xslt#namespace-aliasing?>
  <!-- Purpose: Test basic functionality of namespace-alias. Where XSL elements
       are created by using Literal Result Elements. namespace24 is very similar, but
       it creates the axsl:stylesheet element via xsl:element. -->

<xsl:namespace-alias stylesheet-prefix="axsl" result-prefix="xsl"/>

<xsl:template match="/">
  <axsl:stylesheet version="1.0">
    <xsl:apply-templates/>
  </axsl:stylesheet>
</xsl:template>

<xsl:template match="block">
  <axsl:template match="{.}">
     <axsl:apply-templates/>
  </axsl:template>
</xsl:template>

</xsl:stylesheet>
