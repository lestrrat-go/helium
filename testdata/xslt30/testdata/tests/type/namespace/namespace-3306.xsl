<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns="http://default.com/">

<?spec xslt#creating-attributes?>
    <!-- Purpose: See what happens to attribute when default namespace is declared. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:element name="inner">
      <xsl:attribute name="Attr1">true</xsl:attribute>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>