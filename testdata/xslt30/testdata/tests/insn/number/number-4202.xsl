<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:b="http://z.test.com/" exclude-result-prefixes="b">


  <!-- Purpose: Namespaced elements should work just like non-namespaced ones even when prefixes in source document and stylesheet are different. -->


<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="b:doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="b:note">
  <xsl:element name="bnote">
    <xsl:attribute name="number">
      <xsl:number level="single" from="b:chapter"/>
    </xsl:attribute>
    <xsl:value-of select="."/>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
