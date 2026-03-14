<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:z="http://z.test.com/" exclude-result-prefixes="z">

 
  <!-- Purpose: If the source document has a named default namespace, the counting (on xsl:number) should still work. -->
  

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="z:doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="z:note">
  <xsl:element name="znote">
    <xsl:attribute name="number">
      <xsl:number level="single" from="z:chapter"/>
    </xsl:attribute>
    <xsl:value-of select="."/>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
