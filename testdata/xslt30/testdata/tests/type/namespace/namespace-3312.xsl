<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:pfix="http://party.com/">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Declare prefixed namespace at stylesheet level, then use prefix in attribute
    name only. No namespace nor xmlns on xsl:attribute itself. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:element name="inner">
      <xsl:attribute name="pfix:nuts">pecan</xsl:attribute>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>